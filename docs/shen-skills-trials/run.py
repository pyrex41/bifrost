#!/usr/bin/env python3
"""Run paired exploratory skill trials; never interpret agent exit 0 as task success."""
import argparse
import concurrent.futures
import hashlib
import json
import os
from pathlib import Path
import shutil
import signal
import subprocess
import time


def manifest(root):
    return {str(p.relative_to(root)): hashlib.sha256(p.read_bytes()).hexdigest()
            for p in sorted(root.rglob('*')) if p.is_file()}


def run_cell(cell, model, args, prompt):
    env = dict(os.environ, BIFROST_TRIAL_BIN=str(args.checker),
               BIFROST_SHEN_JIT=str(args.runtime), BIFROST_PORTS_ROOT=str(args.ports))
    command = ['pi', '--provider', args.provider, '--model', model,
               '--thinking', 'high', '--no-extensions', '--no-skills',
               '--no-prompt-templates', '--no-context-files',
               '--session-dir', str(cell / 'sessions'), '--print', '@' + str(prompt)]
    metadata = {'model': model, 'provider': args.provider, 'command': command,
                'started_unix': time.time(), 'budget_seconds': args.seconds,
                'input_sha256': manifest(cell / 'input-skills'),
                'prompt_sha256': hashlib.sha256(prompt.read_bytes()).hexdigest(),
                'checker_sha256': hashlib.sha256(args.checker.read_bytes()).hexdigest(),
                'runtime_sha256': hashlib.sha256(args.runtime.read_bytes()).hexdigest()}
    with (cell / 'agent.log').open('w') as log:
        process = subprocess.Popen(command, cwd=cell, env=env, stdout=log,
                                   stderr=subprocess.STDOUT, start_new_session=True)
        try:
            metadata['exit_code'] = process.wait(timeout=args.seconds)
            metadata['timed_out'] = False
        except subprocess.TimeoutExpired:
            os.killpg(process.pid, signal.SIGTERM)
            try:
                process.wait(timeout=10)
            except subprocess.TimeoutExpired:
                os.killpg(process.pid, signal.SIGKILL)
                process.wait()
            metadata.update(exit_code=process.returncode, timed_out=True)
    metadata['elapsed_seconds'] = round(time.time() - metadata['started_unix'], 2)
    metadata['inputs_unchanged'] = metadata['input_sha256'] == manifest(cell / 'input-skills')
    metadata['answers_sha256'] = manifest(cell / 'answers')
    (cell / 'metadata.json').write_text(json.dumps(metadata, indent=2) + '\n')
    return cell.name, metadata['exit_code'], metadata['timed_out']


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--repo', type=Path, required=True)
    parser.add_argument('--baseline', default='51a8f0d')
    parser.add_argument('--checker', type=Path, required=True)
    parser.add_argument('--runtime', type=Path, required=True)
    parser.add_argument('--ports', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--provider', default='openai-codex')
    parser.add_argument('--models', nargs='+', default=['gpt-6.1-sol', 'gpt-6.1-terra'])
    parser.add_argument('--seconds', type=int, default=600)
    parser.add_argument('--jobs', type=int, default=2)
    args = parser.parse_args()
    for name in ('repo', 'checker', 'runtime', 'ports', 'output'):
        setattr(args, name, getattr(args, name).resolve())
    if args.seconds <= 0 or args.jobs <= 0:
        parser.error('seconds and jobs must be positive')
    args.output.mkdir(parents=True, exist_ok=False)
    snapshots = args.output / 'snapshots'
    baseline = snapshots / 'original'
    baseline.mkdir(parents=True)
    archive = subprocess.run(['git', '-C', str(args.repo), 'archive', args.baseline, 'skills'],
                             check=True, stdout=subprocess.PIPE).stdout
    subprocess.run(['tar', '-x', '-C', str(baseline)], input=archive, check=True)
    shutil.copytree(args.repo / 'skills', snapshots / 'revised' / 'skills')
    prompt = args.output / 'tasks.md'
    shutil.copyfile(Path(__file__).with_name('tasks.md'), prompt)
    cells = []
    for model in args.models:
        for condition in ('original', 'revised'):
            cell = args.output / (model + '-' + condition)
            cell.mkdir()
            shutil.copytree(snapshots / condition / 'skills', cell / 'input-skills')
            cells.append((cell, model))
    with concurrent.futures.ThreadPoolExecutor(max_workers=args.jobs) as executor:
        futures = [executor.submit(run_cell, cell, model, args, prompt) for cell, model in cells]
        for future in concurrent.futures.as_completed(futures):
            print(json.dumps(future.result()), flush=True)


if __name__ == '__main__':
    main()
