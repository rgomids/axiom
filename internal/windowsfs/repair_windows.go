package windowsfs

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

// RepairTarget is one explicitly selected Runtime integration directory.
// Parents need replacement protection; skill roots also need private access.
type RepairTarget struct {
	Path    string
	Private bool
}

type RepairChange struct {
	Path     string `json:"path"`
	Identity string `json:"identity"`
	Before   string `json:"before"`
	After    string `json:"after"`
	Private  bool   `json:"private"`
}

// RepairPlan pins every existing ancestor with a non-delete-sharing handle.
// It never walks descendants or modifies objects not explicitly in Targets.
type RepairPlan struct {
	Changes []RepairChange `json:"changes"`
	Digest  string         `json:"digest"`
	files   map[string]*os.File
	checks  []RepairTarget
}

func (p *RepairPlan) Close() {
	for _, f := range p.files {
		f.Close()
	}
	p.files = nil
}

func repairSecurity(f *os.File) (*windows.SECURITY_DESCRIPTOR, error) {
	sd, err := windows.GetSecurityInfo(windows.Handle(f.Fd()), windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION|windows.GROUP_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
	if err != nil || sd == nil || !sd.IsValid() {
		return nil, fmt.Errorf("%w: cannot inspect repair descriptor: %v", ErrUnsafe, err)
	}
	return sd, nil
}

// repairedDescriptor retains owner, group, deny ACEs and trusted allow ACEs.
// It only subtracts permissions rejected by Check from untrusted allow ACEs.
func repairedDescriptor(sd *windows.SECURITY_DESCRIPTOR, private bool) (string, error) {
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return "", err
	}
	owner, _, err := sd.Owner()
	if err != nil || owner == nil || !currentOwner(owner, user.User.Sid) {
		return "", fmt.Errorf("%w: repair requires current ownership", ErrUnsafe)
	}
	acl, _, err := sd.DACL()
	if err != nil || acl == nil {
		return "", fmt.Errorf("%w: repair requires explicit DACL", ErrUnsafe)
	}
	mask := uint32(windows.DELETE | windows.WRITE_DAC | windows.WRITE_OWNER | fileDeleteChild | windows.GENERIC_WRITE | windows.GENERIC_ALL)
	if private {
		mask |= windows.FILE_READ_DATA | windows.FILE_WRITE_DATA | windows.FILE_APPEND_DATA | windows.FILE_READ_EA | windows.FILE_WRITE_EA | windows.FILE_WRITE_ATTRIBUTES | windows.FILE_EXECUTE | windows.GENERIC_READ | windows.GENERIC_EXECUTE
	}
	// The native ACL header is eight bytes; copy complete ACEs to retain flags,
	// ordering and deny semantics. Unknown ACE types are never transformed.
	data := append([]byte(nil), unsafe.Slice((*byte)(unsafe.Pointer(acl)), 8)...)
	count := uint16(0)
	for i := uint32(0); i < uint32(acl.AceCount); i++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if windows.GetAce(acl, i, &ace) != nil || ace == nil || ace.Header.AceSize < 12 {
			return "", ErrUnsafe
		}
		if ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE && ace.Header.AceType != windows.ACCESS_DENIED_ACE_TYPE {
			return "", fmt.Errorf("%w: unsupported repair ACE", ErrUnsafe)
		}
		sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
		if !sid.IsValid() {
			return "", ErrUnsafe
		}
		wire := append([]byte(nil), unsafe.Slice((*byte)(unsafe.Pointer(ace)), int(ace.Header.AceSize))...)
		if ace.Header.AceType == windows.ACCESS_ALLOWED_ACE_TYPE && !sid.Equals(user.User.Sid) && !privileged(sid) {
			// Inherit-only ACEs are retained on ancestors; protecting the leaf
			// prevents unsafe ancestor grants entering newly created skills.
			if private || ace.Header.AceFlags&windows.INHERIT_ONLY_ACE == 0 {
				remaining := uint32(ace.Mask) &^ mask
				if remaining == 0 {
					continue
				}
				binary.LittleEndian.PutUint32(wire[4:8], remaining)
			}
		}
		data = append(data, wire...)
		count++
	}
	if len(data) > 65535 {
		return "", ErrUnsafe
	}
	binary.LittleEndian.PutUint16(data[2:4], uint16(len(data)))
	binary.LittleEndian.PutUint16(data[4:6], count)
	absolute, err := sd.ToAbsolute()
	if err != nil {
		return "", err
	}
	if err := absolute.SetDACL((*windows.ACL)(unsafe.Pointer(&data[0])), true, false); err != nil {
		return "", err
	}
	if err := absolute.SetControl(windows.SE_DACL_PROTECTED|windows.SE_DACL_AUTO_INHERITED|windows.SE_DACL_AUTO_INHERIT_REQ, windows.SE_DACL_PROTECTED); err != nil {
		return "", err
	}
	result := absolute.String()
	runtime.KeepAlive(data)
	if result == "" {
		return "", ErrUnsafe
	}
	return result, nil
}

// PreviewRepair permits only the standard Runtime configuration parents and
// skill roots under this profile. It refuses unsafe ancestors outside that set.
func PreviewRepair(home string, targets []RepairTarget) (*RepairPlan, error) {
	home, err := Canonical(home)
	if err != nil {
		return nil, err
	}
	allowed := map[string]bool{}
	for _, name := range []string{".agents", ".claude"} {
		allowed[strings.ToLower(filepath.Join(home, name))] = false
		allowed[strings.ToLower(filepath.Join(home, name, "skills"))] = true
	}
	repairable := map[string]bool{}
	paths := map[string]string{}
	for _, target := range targets {
		path, err := Canonical(target.Path)
		if err != nil {
			return nil, err
		}
		key := strings.ToLower(path)
		mode, ok := allowed[key]
		if !ok || mode != target.Private {
			return nil, fmt.Errorf("%w: repair outside standard Runtime scope", ErrUnsafe)
		}
		repairable[key] = mode
		for current := path; ; current = filepath.Dir(current) {
			paths[strings.ToLower(current)] = current
			if filepath.Dir(current) == current {
				break
			}
		}
	}
	ordered := make([]string, 0, len(paths))
	for _, path := range paths {
		ordered = append(ordered, path)
	}
	sort.Slice(ordered, func(i, j int) bool {
		if len(ordered[i]) != len(ordered[j]) {
			return len(ordered[i]) < len(ordered[j])
		}
		return ordered[i] < ordered[j]
	})
	p := &RepairPlan{Changes: []RepairChange{}, files: map[string]*os.File{}}
	fail := func(err error) (*RepairPlan, error) { p.Close(); return nil, err }
	for _, path := range ordered {
		private, canRepair := repairable[strings.ToLower(path)]
		name, err := windows.UTF16PtrFromString(path)
		if err != nil {
			return fail(err)
		}
		access := uint32(windows.READ_CONTROL | windows.FILE_READ_ATTRIBUTES)
		h, err := windows.CreateFile(name, access, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
		if errors.Is(err, windows.ERROR_FILE_NOT_FOUND) || errors.Is(err, windows.ERROR_PATH_NOT_FOUND) {
			continue
		}
		if err != nil {
			return fail(err)
		}
		f := os.NewFile(uintptr(h), path)
		p.files[path] = f
		var info windows.ByHandleFileInformation
		if err := windows.GetFileInformationByHandle(h, &info); err != nil || info.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 || info.FileAttributes&windows.FILE_ATTRIBUTE_DIRECTORY == 0 {
			return fail(ErrUnsafe)
		}
		p.checks = append(p.checks, RepairTarget{path, private})
		if err := Check(f, private); err == nil {
			continue
		} else if !canRepair {
			return fail(err)
		}
		// Safe roots require only inspection access. Request WRITE_DAC solely
		// for a directory actually requiring repair, while its original handle
		// still prevents replacement; bind the new handle to the same object.
		originalIdentity, err := Identity(f)
		if err != nil {
			return fail(err)
		}
		writable, err := windows.CreateFile(name, access|windows.WRITE_DAC, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
		if err != nil {
			return fail(err)
		}
		repairFile := os.NewFile(uintptr(writable), path)
		writableIdentity, err := Identity(repairFile)
		if err != nil || writableIdentity != originalIdentity {
			repairFile.Close()
			return fail(ErrUnsafe)
		}
		f.Close()
		f = repairFile
		p.files[path] = f
		sd, err := repairSecurity(f)
		if err != nil {
			return fail(err)
		}
		after, err := repairedDescriptor(sd, private)
		if err != nil {
			return fail(err)
		}
		identity, err := Identity(f)
		if err != nil {
			return fail(err)
		}
		p.Changes = append(p.Changes, RepairChange{path, identity, sd.String(), after, private})
	}
	wire, err := json.Marshal(p.Changes)
	if err != nil {
		return fail(err)
	}
	digest := sha256.Sum256(wire)
	p.Digest = hex.EncodeToString(digest[:])
	return p, nil
}

func setRepairDescriptor(f *os.File, descriptor string) error {
	sd, err := windows.SecurityDescriptorFromString(descriptor)
	if err != nil {
		return err
	}
	acl, _, err := sd.DACL()
	if err != nil || acl == nil {
		return ErrUnsafe
	}
	control, _, err := sd.Control()
	if err != nil {
		return err
	}
	flags := windows.SECURITY_INFORMATION(windows.DACL_SECURITY_INFORMATION | windows.UNPROTECTED_DACL_SECURITY_INFORMATION)
	if control&windows.SE_DACL_PROTECTED != 0 {
		flags = windows.SECURITY_INFORMATION(windows.DACL_SECURITY_INFORMATION | windows.PROTECTED_DACL_SECURITY_INFORMATION)
	}
	// SetSecurityInfo normalizes inherited ACEs and can propagate to children.
	// The native handle operation sets this object's descriptor only, retaining
	// the original inheritance flags needed for exact recovery.
	status, _, _ := windows.NewLazySystemDLL("ntdll.dll").NewProc("NtSetSecurityObject").Call(uintptr(f.Fd()), uintptr(flags), uintptr(unsafe.Pointer(sd)))
	runtime.KeepAlive(sd)
	if windows.NTStatus(status) != 0 {
		return windows.NTStatus(status).Errno()
	}
	return nil
}

// Windows clears DACL_AUTO_INHERITED bookkeeping on direct object updates.
// Recovery requires identical ACEs, owner/group and protection; only that
// non-access bookkeeping bit is ignored.
func equivalentRepairDescriptor(left, right string) bool {
	normalize := func(text string) string {
		start := strings.Index(text, "D:")
		if start < 0 {
			return text
		}
		end := strings.Index(text[start:], "(")
		if end < 0 {
			return text
		}
		end += start
		return text[:start] + strings.ReplaceAll(text[start:end], "AI", "") + text[end:]
	}
	return normalize(left) == normalize(right)
}

// Apply requires the exact preview digest and a durable private backup before
// the first ACL effect. Any failed effect triggers rollback over pinned handles.
func (p *RepairPlan) Apply(authority string, backup func([]byte) error) error {
	if authority != p.Digest || backup == nil || p.files == nil {
		return fmt.Errorf("%w: repair authority required", ErrUnsafe)
	}
	for _, change := range p.Changes {
		f := p.files[change.Path]
		identity, err := Identity(f)
		if err != nil || identity != change.Identity {
			return ErrUnsafe
		}
		sd, err := repairSecurity(f)
		if err != nil || sd.String() != change.Before {
			return fmt.Errorf("%w: repair descriptor changed", ErrUnsafe)
		}
	}
	wire, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	if err := backup(append(wire, '\n')); err != nil {
		return err
	}
	applied := 0
	rollback := func(cause error) error {
		for i := applied - 1; i >= 0; i-- {
			change := p.Changes[i]
			f := p.files[change.Path]
			if err := setRepairDescriptor(f, change.Before); err != nil {
				cause = errors.Join(cause, fmt.Errorf("rollback incomplete for %q: %w", change.Path, err))
				continue
			}
			sd, err := repairSecurity(f)
			if err != nil || !equivalentRepairDescriptor(sd.String(), change.Before) {
				cause = errors.Join(cause, fmt.Errorf("rollback verification failed for %q", change.Path))
			}
		}
		return cause
	}
	for _, change := range p.Changes {
		// Recheck immediately before each mutation as well as before backup.
		sd, err := repairSecurity(p.files[change.Path])
		if err != nil || sd.String() != change.Before {
			return rollback(ErrUnsafe)
		}
		applied++ // Include the current object even if its API result is uncertain.
		if err := setRepairDescriptor(p.files[change.Path], change.After); err != nil {
			return rollback(err)
		}
		if err := Check(p.files[change.Path], change.Private); err != nil {
			return rollback(err)
		}
		confirmed, err := repairSecurity(p.files[change.Path])
		if err != nil || confirmed.String() != change.After {
			return rollback(fmt.Errorf("%w: repair postcondition changed", ErrUnsafe))
		}
	}
	for _, target := range p.checks {
		if err := Check(p.files[target.Path], target.Private); err != nil {
			return rollback(err)
		}
	}
	return nil
}

// RestoreRepair restores exactly the captured descriptors, only while the
// same objects still have either the approved post-repair or original ACL.
// It never overwrites a later permission change.
func RestoreRepair(home string, wire []byte, authority string) error {
	if len(wire) > 65536 {
		return ErrUnsafe
	}
	var saved struct {
		Changes []RepairChange `json:"changes"`
		Digest  string         `json:"digest"`
	}
	if err := json.Unmarshal(wire, &saved); err != nil {
		return err
	}
	canonical, err := json.Marshal(saved.Changes)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(canonical)
	if len(saved.Changes) == 0 || len(saved.Changes) > 4 || saved.Digest != hex.EncodeToString(digest[:]) || authority != saved.Digest {
		return fmt.Errorf("%w: recovery authority required", ErrUnsafe)
	}
	targets := make([]RepairTarget, 0, len(saved.Changes))
	seen := map[string]bool{}
	for _, change := range saved.Changes {
		key := strings.ToLower(change.Path)
		if seen[key] {
			return fmt.Errorf("%w: duplicate recovery target", ErrUnsafe)
		}
		seen[key] = true
		targets = append(targets, RepairTarget{change.Path, change.Private})
	}
	plan, err := PreviewRepair(home, targets)
	if err != nil {
		return err
	}
	defer plan.Close()
	for _, change := range saved.Changes {
		f := plan.files[change.Path]
		if f == nil {
			return ErrUnsafe
		}
		identity, err := Identity(f)
		if err != nil || identity != change.Identity {
			return fmt.Errorf("%w: recovery object changed", ErrUnsafe)
		}
		sd, err := repairSecurity(f)
		if err != nil || (!equivalentRepairDescriptor(sd.String(), change.Before) && !equivalentRepairDescriptor(sd.String(), change.After)) {
			return fmt.Errorf("%w: recovery descriptor changed", ErrUnsafe)
		}
		// Safe post-repair roots were opened for inspection only. Request write
		// access only for the exact approved recovery object, still pinned.
		name, err := windows.UTF16PtrFromString(change.Path)
		if err != nil {
			return err
		}
		h, err := windows.CreateFile(name, windows.READ_CONTROL|windows.FILE_READ_ATTRIBUTES|windows.WRITE_DAC, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
		if err != nil {
			return err
		}
		writable := os.NewFile(uintptr(h), change.Path)
		actual, err := Identity(writable)
		if err != nil || actual != change.Identity {
			writable.Close()
			return ErrUnsafe
		}
		f.Close()
		plan.files[change.Path] = writable
	}
	for i := len(saved.Changes) - 1; i >= 0; i-- {
		change := saved.Changes[i]
		f := plan.files[change.Path]
		if err := setRepairDescriptor(f, change.Before); err != nil {
			return err
		}
		sd, err := repairSecurity(f)
		if err != nil || !equivalentRepairDescriptor(sd.String(), change.Before) {
			return fmt.Errorf("%w: recovery verification failed", ErrUnsafe)
		}
	}
	return nil
}
