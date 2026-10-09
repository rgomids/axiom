#!/usr/bin/env python3
"""Native prepared-byte installation lifecycle (#256), no builds or Providers.

The baseline is the genuine published v0.10.0 distribution. Reports and logs
must be outside the immutable artifact directories. Only native Linux runs.
"""
import argparse
import importlib.util
import json
import os
from pathlib import Path
import platform
import shutil
import subprocess
import sys
import tarfile
import tempfile

sys.dont_write_bytecode = True
spec = importlib.util.spec_from_file_location("smoke", Path(__file__).with_name("smoke-prepared-artifact.py"))
smoke = importlib.util.module_from_spec(spec)
spec.loader.exec_module(smoke)


def require(condition, message):
    if not condition:
        raise ValueError(message)


def extract(root, version, row, destination):
    name = f"axiom-{version}-{row}"
    archive = root / f"{name}.tar.gz"
    checksum = smoke.digest(archive)
    require((root / "SHA256SUMS").read_text().splitlines().count(f"{checksum}  {archive.name}") == 1,
            "archive checksum mismatch")
    with tarfile.open(archive, "r:gz") as source:
        members = source.getmembers()
        require(len(members) <= 256 and sum(m.size for m in members) <= 128 * 1024 * 1024,
                "archive exceeds acceptance bound")
        names = set()
        for member in members:
            parts = member.name.split("/")
            require(parts[0] == name and all(p not in ("", ".", "..") for p in parts)
                    and "\\" not in member.name and (member.isfile() or member.isdir())
                    and member.name not in names, "unsafe archive entry")
            names.add(member.name)
        source.extractall(destination, members=members)
    bundle = destination / name
    metadata = dict(line.split("=", 1) for line in (bundle / "release-metadata.txt").read_text().splitlines())
    require(metadata["version"] == version and metadata["release"] == "true"
            and metadata["sourceState"] == "clean", "release metadata mismatch")
    return bundle, archive, checksum, metadata


def acceptance(args, report):
    require((platform.system(), platform.machine()) == ("Linux", {"linux-arm64": "aarch64", "linux-amd64": "x86_64"}[args.row]),
            "native host does not match row")
    roots = [Path(args.directory), Path(args.previous)]
    require(all(p.is_absolute() and p.is_dir() and not p.is_symlink() for p in roots),
            "absolute regular artifact directories required")
    before = [smoke.inventory(p) for p in roots]
    try:
        # Preserve #244's exact provenance/lifecycle and input immutability checks.
        smoke.smoke(args, report)
        with tempfile.TemporaryDirectory(prefix="axiom-native-") as temporary:
            work = Path(temporary).resolve()
            extracted = work / "bundles"
            extracted.mkdir()
            candidate, archive, checksum, metadata = extract(roots[0], args.version, args.row, extracted)
            previous, prior_archive, prior_checksum, prior_metadata = extract(roots[1], "0.10.0", args.row, extracted)
            report["baseline"] = {"version": "0.10.0", "archiveSha256": prior_checksum,
                                  "revision": prior_metadata["revision"]}
            report["archiveSha256"] = checksum
            log = Path(args.log)
            log.parent.mkdir(parents=True, exist_ok=True)
            with log.open("w") as diagnostics:
                def command(argv, env, cwd, expected=0):
                    result = subprocess.run([str(a) for a in argv], env=env, cwd=cwd,
                                            stdin=subprocess.DEVNULL, capture_output=True,
                                            timeout=60, check=False)
                    diagnostics.write(json.dumps({"command": [str(a) for a in argv],
                                                  "exit": result.returncode}) + "\n")
                    diagnostics.write(result.stdout.decode(errors="replace")[:65536] + "\n")
                    diagnostics.write(result.stderr.decode(errors="replace")[:65536] + "\n")
                    diagnostics.flush()
                    require(result.returncode == expected if expected is not None else result.returncode != 0,
                            "unexpected command exit: " + " ".join(str(a) for a in argv[:3]))
                    require(len(result.stdout) <= 65536, "CLI output exceeds acceptance bound")
                    return result.stdout.decode()

                def scenario(identifier, action):
                    entry = {"id": identifier, "expected": "pass", "result": "fail"}
                    report["scenarios"].append(entry)
                    action()
                    entry["result"] = "pass"

                def environment(name):
                    home = work / name
                    home.mkdir(mode=0o700)
                    for directory in ("cwd", "repository with spaces", "tmp"):
                        (home / directory).mkdir(mode=0o700)
                    return home, {"HOME": str(home), "USERPROFILE": str(home),
                                  "PATH": f"{home / 'bin'}:/usr/bin:/bin",
                                  "TMPDIR": str(home / "tmp"), "LANG": "C", "LC_ALL": "C",
                                  "LINGO_PROJECTS_ROOT": str(home / "projects"),
                                  "LINGO_STATE_ROOT": str(home / "state"),
                                  "AXIOM_CODEX_SKILLS_ROOT": str(home / "skills"),
                                  "AXIOM_GH_BIN": str(home / "unavailable-gh")}

                home, env = environment("home with spaces")
                binary = home / "bin" / "axiom"
                receipt = home / "receipt"

                def install(bundle, subject, checksums=None, extra=None, target=home, expected=0):
                    installer_env = dict(env, HOME=str(target), USERPROFILE=str(target))
                    installer_env.update(extra or {})
                    return command(["/bin/bash", bundle / "install.sh", "--archive", subject,
                                    "--checksums", checksums or roots[0] / "SHA256SUMS",
                                    "--bin-dir", target / "bin", "--receipt-dir", target / "receipt"],
                                   installer_env, home / "cwd", expected)

                def cli(*argv, expected=0):
                    output = command(["axiom", "--json", *argv], env, home / "cwd", expected)
                    event = json.loads(output)
                    require(event.get("status") == "success" if expected == 0 else event.get("status") != "success",
                            "unexpected CLI status")
                    return event

                def version(expected, revision):
                    require(cli("version")["provenance"] == {"product": "Axiom", "version": expected,
                            "revision": revision[:12], "sourceState": "clean"}, "installed provenance mismatch")

                scenario("fresh-install", lambda: require("install_status=installed" in install(candidate, archive), "fresh status"))
                scenario("discovery-provenance", lambda: version(args.version, args.revision))
                scenario("private-permissions", lambda: require(binary.stat().st_mode & 0o777 == 0o700
                         and (receipt / "installation.receipt").stat().st_mode & 0o777 == 0o600, "unsafe permissions"))
                configuration = ("project", "configure", "--slug", "native", "--name", "Native",
                                 "--repository", f"main={home / 'repository with spaces'}", "--work-item-provider", "none")

                def configure():
                    setup = cli(*configuration)["setup"]
                    cli(*configuration, "--project-id", setup["projectId"], "--preview-digest", setup["digest"], "--authorize-local")

                def readable():
                    require(cli("project", "show", "--selector", "native")["project"]["slug"] == "native", "project unreadable")
                    require(any(p["slug"] == "native" for p in cli("project", "list")["projects"]), "project not listed")

                scenario("configuration-cli", lambda: (configure(), readable()))
                state = smoke.inventory(home / "projects"), smoke.inventory(home / "state")
                owned = smoke.digest(binary), smoke.inventory(receipt)
                scenario("reinstall-idempotent", lambda: require("install_status=unchanged" in install(candidate, archive)
                         and owned == (smoke.digest(binary), smoke.inventory(receipt))
                         and sorted(p.name for p in binary.parent.iterdir()) == ["axiom"]
                         and state == (smoke.inventory(home / "projects"), smoke.inventory(home / "state")), "reinstall changed state"))
                scenario("invalid-arguments", lambda: cli("project", "show", "--unsupported", expected=None))
                scenario("missing-project", lambda: cli("project", "show", "--selector", "absent", expected=None))

                def malformed():
                    manifests = list((home / "projects").rglob("axiom.yaml"))
                    require(len(manifests) == 1, "one project manifest required")
                    manifest = manifests[0]
                    saved = manifest.read_bytes()
                    manifest.write_text("invalid: [\n")
                    try:
                        cli("project", "show", "--selector", "native", expected=None)
                        require(manifest.read_text() == "invalid: [\n", "invalid state overwritten")
                    finally:
                        manifest.write_bytes(saved)
                    readable()

                scenario("malformed-config-recovery", malformed)

                def missing_configuration():
                    manifests = list((home / "projects").rglob("axiom.yaml"))
                    require(len(manifests) == 1, "one project manifest required")
                    manifest = manifests[0]
                    saved = manifest.read_bytes()
                    manifest.unlink()
                    try:
                        cli("project", "show", "--selector", "native", expected=None)
                        require(not manifest.exists(), "missing config silently recreated")
                    finally:
                        manifest.write_bytes(saved)
                        manifest.chmod(0o600)
                    readable()

                scenario("missing-config-recovery", missing_configuration)
                bad = work / archive.name
                shutil.copyfile(archive, bad)
                with bad.open("ab") as stream:
                    stream.write(b"corrupt")
                scenario("corrupt-artifact-refused", lambda: (install(candidate, bad, expected=None),
                         require(owned == (smoke.digest(binary), smoke.inventory(receipt)), "corruption mutated install")))
                foreign = work / "foreign"
                foreign.mkdir(mode=0o700)
                (foreign / "bin").mkdir(mode=0o700)
                (foreign / "bin" / "axiom").write_text("foreign")
                scenario("foreign-binary-preserved", lambda: (install(candidate, archive, target=foreign, expected=None),
                         require((foreign / "bin" / "axiom").read_text() == "foreign", "foreign binary overwritten")))
                link = work / "linked"
                link.symlink_to(home, target_is_directory=True)
                scenario("symlink-destination-refused", lambda: (install(candidate, archive, target=link, expected=None),
                         require(owned == (smoke.digest(binary), smoke.inventory(receipt)), "symlink mutated install")))

                # A pre-binary interruption may be safely retried; after-binary
                # fresh-install recovery intentionally requires guided recovery.
                interrupted = work / "interrupted"
                interrupted.mkdir(mode=0o700)
                scenario("install-interruption-recovery", lambda: (
                    install(candidate, archive, target=interrupted, extra={"AXIOM_INSTALL_TEST_FAIL_STAGE": "before_binary"}, expected=75),
                    require(not (interrupted / "bin" / "axiom").exists(), "interruption published binary"),
                    require("install_status=installed" in install(candidate, archive, target=interrupted), "retry failed")))

                # Separate real prior installation with its own configured state.
                home, env = environment("upgrade home")
                binary = home / "bin" / "axiom"
                receipt = home / "receipt"
                configuration = ("project", "configure", "--slug", "native", "--name", "Native",
                                 "--repository", f"main={home / 'repository with spaces'}", "--work-item-provider", "none")

                def upgrade():
                    install(previous, prior_archive, roots[1] / "SHA256SUMS", target=home)
                    version("0.10.0", prior_metadata["revision"])
                    configure()
                    persisted = smoke.inventory(home / "projects"), smoke.inventory(home / "state")
                    previous_owned = smoke.digest(binary), smoke.inventory(receipt)
                    install(candidate, bad, target=home, expected=None)
                    require(previous_owned == (smoke.digest(binary), smoke.inventory(receipt))
                            and persisted == (smoke.inventory(home / "projects"), smoke.inventory(home / "state")),
                            "failed upgrade changed previous installation or state")
                    readable()
                    require("install_status=upgraded" in install(candidate, archive, target=home), "upgrade status")
                    version(args.version, args.revision)
                    require(persisted == (smoke.inventory(home / "projects"), smoke.inventory(home / "state")), "upgrade mutated persisted state")
                    readable()
                    require("install_status=unchanged" in install(candidate, archive, target=home), "upgrade reinstall status")
                    compatibility = cli("compatibility", "inspect")
                    require(compatibility.get("maintenance", {}).get("classification") == "valid_v1",
                            "state compatibility not valid_v1")

                scenario("genuine-prior-upgrade-state-reinstall", upgrade)
    finally:
        report["inputsUnchanged"] = all(smoke.inventory(p) == snapshot for p, snapshot in zip(roots, before))
        require(report["inputsUnchanged"], "prepared or baseline inputs changed")


def main():
    os.umask(0o077)
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--dir", dest="directory", required=True)
    parser.add_argument("--previous", required=True)
    parser.add_argument("--version", required=True)
    parser.add_argument("--revision", required=True)
    parser.add_argument("--row", choices=("linux-arm64", "linux-amd64"), required=True)
    parser.add_argument("--log", required=True)
    args = parser.parse_args()
    report = {"schema": "axiom-native-acceptance/v1", "version": args.version,
              "revision": args.revision, "row": args.row, "result": "fail", "scenarios": [],
              "environment": {"os": platform.system(), "architecture": platform.machine(),
                              "platform": platform.platform(), "runnerImage": os.getenv("ImageOS", "local"),
                              "runnerImageVersion": os.getenv("ImageVersion", "unknown")},
              "workflowRun": f"{os.getenv('GITHUB_SERVER_URL', '')}/{os.getenv('GITHUB_REPOSITORY', '')}/actions/runs/{os.getenv('GITHUB_RUN_ID', '')}",
              "job": os.getenv("GITHUB_JOB", "local")}
    try:
        log = Path(args.log).resolve()
        require(all(not log.is_relative_to(Path(p).resolve()) for p in (args.directory, args.previous)),
                "log must be outside immutable inputs")
        acceptance(args, report)
        report["result"] = "pass"
    except (ValueError, OSError, KeyError, TypeError, AttributeError, tarfile.TarError, subprocess.TimeoutExpired) as error:
        report["error"] = str(error)
    print(json.dumps(report, sort_keys=True))
    return 0 if report["result"] == "pass" else 1


if __name__ == "__main__":
    raise SystemExit(main())
