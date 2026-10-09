#!/usr/bin/env python3
"""Offline documentation fixture checks, not a product workflow implementation.

Uses only the standard library. Supports exactly the Schema keywords present in
workflow.schema.json and ASCII/integer JCS fixtures; rejects unsupported keywords
rather than pretending to be a general JSON Schema or RFC 8785 implementation.
"""

import copy
import hashlib
import json
from pathlib import Path
import re

ROOT = Path(__file__).resolve().parent
PHASES = ['intake', 'specification', 'planning', 'implementation', 'review', 'completion']
KEYWORDS = {'$schema', '$id', 'title', 'type', 'additionalProperties', 'required',
            'properties', 'items', 'minItems', 'maxItems', 'minLength', 'maxLength',
            'pattern', 'minimum', 'maximum', 'enum', 'const'}


def require(condition, message):
    if not condition:
        raise ValueError(message)


def pairs(items):
    result = {}
    for key, value in items:
        require(key not in result, f'duplicate key: {key}')
        result[key] = value
    return result


def read(path):
    return json.loads(path.read_text(encoding='utf-8'), object_pairs_hook=pairs)


def structural(value, schema, path='$'):
    require(set(schema) <= KEYWORDS, f'{path}: unsupported schema keyword')
    kind = schema['type']
    types = {'object': dict, 'array': list, 'string': str, 'integer': int, 'boolean': bool}
    require(type(value) is types[kind], f'{path}: expected {kind}')
    if 'const' in schema:
        require(value == schema['const'], f'{path}: const')
    if 'enum' in schema:
        require(value in schema['enum'], f'{path}: enum')
    if kind == 'object':
        require(schema['additionalProperties'] is False, f'{path}: open object')
        require(set(value) <= set(schema['properties']), f'{path}: unknown property')
        require(set(schema['required']) <= set(value), f'{path}: required property')
        for key, item in value.items():
            structural(item, schema['properties'][key], f'{path}.{key}')
    elif kind == 'array':
        require(schema['minItems'] <= len(value) <= schema['maxItems'], f'{path}: list bound')
        for index, item in enumerate(value):
            structural(item, schema['items'], f'{path}[{index}]')
    elif kind == 'string':
        require(schema.get('minLength', 0) <= len(value) <= schema.get('maxLength', 16384), f'{path}: length')
        require(len(value.encode('utf-8')) <= schema.get('maxLength', 16384), f'{path}: byte bound')
        if 'pattern' in schema:
            require(re.fullmatch(schema['pattern'], value) is not None, f'{path}: token')
    elif kind == 'integer':
        require(schema.get('minimum', 0) <= value <= schema.get('maximum', 2147483647), f'{path}: integer bound')


def unique(items, key, where):
    values = [item[key] for item in items]
    require(len(set(values)) == len(values), f'{where}: duplicate {key}')
    return set(values)


def ancestors(nodes, key, visiting=None):
    visiting = set() if visiting is None else visiting
    require(key not in visiting, 'agent dependency cycle')
    require(key in nodes, 'dangling agent dependency')
    result = set()
    for dependency in nodes[key]['dependsOn']:
        result.add(dependency)
        result.update(ancestors(nodes, dependency, visiting | {key}))
    return result


def validate(definition):
    structural(definition, read(ROOT / 'workflow.schema.json'))
    require(len(canonical(definition)) <= 1 << 20, 'definition byte bound')
    stages = definition['stages']
    unique(stages, 'id', 'stages')
    phase_numbers = [PHASES.index(stage['phase']) for stage in stages]
    require(phase_numbers == sorted(phase_numbers) and set(phase_numbers) == set(range(6)), 'phase coverage/order')
    prior_outputs = set()
    seen_gates = set()
    for index, stage in enumerate(stages):
        last_in_phase = index == len(stages) - 1 or stages[index + 1]['phase'] != stage['phase']
        require(stage['checkpoint'] == last_in_phase, 'phase checkpoint')
        inputs = unique(stage['inputs'], 'id', stage['id'])
        outputs = unique(stage['outputs'], 'id', stage['id'])
        validators = unique(stage['validators'], 'id', stage['id'])
        unique(stage['completionCriteria'], 'id', stage['id'])
        for item in stage['inputs']:
            if item['kind'] == 'stage-output':
                require(item['source'] in prior_outputs, 'missing/forward stage input')
        for item in stage['completionCriteria']:
            require(item['outputRef'] in outputs and item['validatorRef'] in validators, 'criterion reference')
        require(len(stage['humanGates']) == len({gate['kind'] for gate in stage['humanGates']}), 'duplicate gate')
        for gate in stage['humanGates']:
            expected = {'planning_authority': ('planning', 'before'),
                        'implementation_authority': ('implementation', 'before'),
                        'review_started': ('review', 'before'),
                        'human_acceptance': ('completion', 'after')}[gate['kind']]
            require((stage['phase'], gate['timing']) == expected, 'gate phase/timing')
            if gate['kind'] == 'human_acceptance':
                require(index == len(stages) - 1, 'acceptance must be final')
            else:
                require(index == 0 or stages[index - 1]['phase'] != stage['phase'], 'gate must precede phase')
            require(gate['kind'] not in seen_gates, 'duplicate workflow gate')
            seen_gates.add(gate['kind'])
        agents = stage['agents']
        unique(agents, 'id', stage['id'])
        require(stage['concurrency'] <= len(agents), 'excess concurrency')
        nodes = {agent['id']: agent for agent in agents}
        owners = [agent for agent in agents if agent['integrationOwner']]
        require(len(owners) == 1 and owners[0]['validationOwner'], 'integration/validation owner')
        owner = owners[0]
        require(ancestors(nodes, owner['id']) == set(nodes) - {owner['id']}, 'disconnected integration')
        assigned_outputs = set()
        for agent in agents:
            require(set(agent['inputs']) <= inputs and set(agent['outputs']) <= outputs, 'agent I/O reference')
            require(len(agent['dependsOn']) == len(set(agent['dependsOn'])), 'duplicate dependency')
            ancestors(nodes, agent['id'])
            effort = agent['effort']
            require((effort['mode'] == 'runtime-default') == (effort['value'] == 'default'), 'effort mode/value')
            assigned_outputs.update(agent['outputs'])
        require({item['id'] for item in stage['outputs'] if item['required']} <= assigned_outputs, 'unowned output')
        if stage['mode'] == 'sequential':
            require(stage['concurrency'] == 1, 'sequential concurrency')
            for agent_index, agent in enumerate(agents):
                if agent_index:
                    require(agents[agent_index - 1]['id'] in agent['dependsOn'], 'sequential chain')
        prior_outputs.update(f"{stage['id']}/{output}" for output in outputs)
    require(seen_gates == {'planning_authority', 'implementation_authority', 'review_started', 'human_acceptance'}, 'protected gate coverage')


def canonical(value):
    # Fixture-only JCS subset: printable ASCII strings, integers, booleans, objects/arrays.
    def supported(item):
        if isinstance(item, str):
            require(all(32 <= ord(char) < 127 for char in item), 'outside fixture JCS string subset')
        elif type(item) in (int, bool):
            return
        elif isinstance(item, list):
            for child in item:
                supported(child)
        elif isinstance(item, dict):
            for key, child in item.items():
                supported(key)
                supported(child)
        else:
            raise ValueError('outside fixture JCS value subset')
    supported(value)
    return json.dumps(value, sort_keys=True, separators=(',', ':'), ensure_ascii=False).encode('utf-8')


def digest(value):
    return hashlib.sha256(canonical(value)).hexdigest()


def set_path(value, path, replacement):
    for key in path[:-1]:
        value = value[key]
    value[path[-1]] = replacement


def main():
    examples = {path.name: read(path) for path in sorted((ROOT / 'examples').glob('*.json'))}
    require(len(examples) == 3, 'fixture inventory')
    for name, value in examples.items():
        validate(value)
        print(f'PASS {name}: sha256:{digest(value)}')
    default = examples['default-sdd-r1.json']
    first, second = examples['custom-r1.json'], examples['custom-r2.json']
    require(all(len(stage['agents']) == 1 for stage in default['stages']), 'default is not single-agent')
    implementation = first['stages'][5]
    require(implementation['agents'][0]['dependsOn'] == implementation['agents'][1]['dependsOn'] == [], 'independent agents')
    require(implementation['agents'][0]['runtimeConstraints'] == ['codex'] and implementation['agents'][1]['runtimeConstraints'] == ['claude'], 'two Runtime example')
    require(digest(first) != digest(second) and first['revision'] + 1 == second['revision'], 'R1/R2 identity')
    reformatted = json.loads(json.dumps(first, sort_keys=True, indent=4))
    require(digest(first) == digest(reformatted), 'formatting changed digest')
    altered = copy.deepcopy(first)
    altered['stages'][5]['instructions'] += ' Changed criterion.'
    require(digest(first) != digest(altered), 'content edit did not change digest')
    expected = read(ROOT / 'example-digests.json')
    require(expected == {name: digest(value) for name, value in examples.items()}, 'frozen example digest mismatch')
    boundary = read(ROOT / 'boundary-examples.json')
    require(boundary['fixtureOnly'] is True, 'boundary examples are not operational evidence')
    for letter, version in [('A', 1), ('B', 2)]:
        binding = boundary[f'execution{letter}Binding']
        selection = boundary[f'projectSelectionR{version}']['workflowSelection']
        require(binding['revision'] == version and binding['digest'] == selection['digest'] == expected[f'custom-r{version}.json'], 'snapshot/selection correlation')
        require(binding['snapshotDigest'] == binding['digest'], 'snapshot digest drift')
    require(boundary['executionABinding']['digest'] != boundary['projectSelectionR2']['workflowSelection']['digest'], 'R1/R2 fixture isolation')
    dependencies = boundary['issueDependencies']
    nodes = {key: {'dependsOn': values} for key, values in dependencies.items()}
    for key in nodes:
        ancestors(nodes, key)
    require(set(nodes) == {'271', '272', '273', '274', '275', '276', '277', '278', '230', '133', '232'}, 'Issue ownership coverage')
    require({'274', '275', '272', '230'} == set(dependencies['276']), 'orchestration joining gates')
    spec = (ROOT / 'spec.md').read_text(encoding='utf-8')
    for number in range(1, 15):
        require(f'WF-{number:03}' in spec and f'AC-{number:03}' in spec, 'requirement/acceptance inventory')
    for number in range(1, 5):
        require(f'HD-{number:03}' in spec, 'human decision inventory')

    cases = [
        ('schema version', ['schemaVersion'], 2),
        ('revision', ['revision'], 0),
        ('unknown property', ['unexpected'], True),
        ('unsafe key', ['workflowId'], '../escape'),
        ('duplicate stage', ['stages', 1, 'id'], 'intake'),
        ('phase order', ['stages', 1, 'phase'], 'review'),
        ('checkpoint', ['stages', 1, 'checkpoint'], True),
        ('forward input', ['stages', 0, 'inputs', 0], {'id':'source','kind':'stage-output','source':'completion/result','required':True}),
        ('unknown validator', ['stages', 0, 'completionCriteria', 0, 'validatorRef'], 'missing'),
        ('unknown output', ['stages', 0, 'completionCriteria', 0, 'outputRef'], 'missing'),
        ('missing gates', ['stages', 3, 'humanGates'], []),
        ('gate phase', ['stages', 0, 'humanGates'], [{'kind':'human_acceptance','timing':'after'}]),
        ('gate timing', ['stages', 3, 'humanGates', 0, 'timing'], 'after'),
        ('instruction bound', ['stages', 0, 'instructions'], 'a' * 16385),
        ('attempt bound', ['stages', 0, 'agents', 0, 'maximumAttempts'], 11),
        ('timeout bound', ['stages', 0, 'agents', 0, 'timeoutSeconds'], 0),
        ('Runtime enum', ['stages', 0, 'agents', 0, 'runtimeConstraints'], ['unknown']),
        ('agent input', ['stages', 0, 'agents', 0, 'inputs'], ['missing']),
        ('agent output', ['stages', 0, 'agents', 0, 'outputs'], ['missing']),
        ('effort mismatch', ['stages', 0, 'agents', 0, 'effort', 'value'], 'high'),
        ('parallel limit', ['stages', 5, 'concurrency'], 4),
        ('dangling dependency', ['stages', 5, 'agents', 0, 'dependsOn'], ['missing']),
        ('self cycle', ['stages', 5, 'agents', 0, 'dependsOn'], ['implementer']),
        ('two-node cycle', ['stages', 5, 'agents', 0, 'dependsOn'], ['integrator']),
        ('missing owner', ['stages', 5, 'agents', 2, 'integrationOwner'], False),
        ('disconnected integration', ['stages', 5, 'agents', 2, 'dependsOn'], ['implementer']),
        ('sequential conflict', ['stages', 5, 'mode'], 'sequential'),
    ]
    for name, path, replacement in cases:
        bad = copy.deepcopy(first)
        set_path(bad, path, replacement)
        try:
            validate(bad)
        except ValueError:
            continue
        raise ValueError(f'negative case accepted: {name}')
    try:
        json.loads('{"revision":1,"revision":2}', object_pairs_hook=pairs)
    except ValueError:
        pass
    else:
        raise ValueError('duplicate JSON key accepted')
    print(f'PASS {len(cases) + 1} negative fixtures; phase/DAG/gate/I/O/limit checks; frozen digest consistency')
    print('PASS boundary R1/R2 correlation; acyclic Issue DAG; requirement/acceptance/decision coverage')
    print('DOCUMENTATION ONLY: no product resume, Runtime, authority, effects or human acceptance proved')


if __name__ == '__main__':
    main()
