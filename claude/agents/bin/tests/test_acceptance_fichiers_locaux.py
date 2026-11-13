"""Acceptance tests, slug fichiers-locaux: LOC-001..003. Boundary: the harness-local command line."""
import os
import subprocess
from pathlib import Path

BIN = [os.environ["HARNESS"], "local"]  # the harness binary under test


def make_repo(tmp_path, origin):
    d = tmp_path / "depot"
    d.mkdir()
    subprocess.run(["git", "init", "-q", str(d)], check=True)
    subprocess.run(["git", "-C", str(d), "remote", "add", "origin", origin], check=True)
    return d


def local_dir(tmp_path, files=None):
    d = tmp_path / "local"
    d.mkdir()
    for name, text in (files or {}).items():
        (d / name).write_text(text)
    return d


def run(local, *args):
    env = dict(os.environ, CLAUDE_LOCAL_DIR=str(local))
    p = subprocess.run([*BIN, *args], env=env, capture_output=True, text=True)
    return p.returncode, p.stdout


def test_LOC001_declared_host_gives_its_forge_tool(tmp_path):
    repo = make_repo(tmp_path, "https://forge.exemple.test/groupe/depot.git")
    local = local_dir(tmp_path, {"forges": "forge.exemple.test glab\n"})
    rc, out = run(local, "forge", "--repo", str(repo))
    assert (rc, out.strip()) == (0, "glab")


def test_LOC002_github_host_is_gh(tmp_path):
    repo = make_repo(tmp_path, "git@github.com:groupe/depot.git")
    rc, out = run(local_dir(tmp_path), "forge", "--repo", str(repo))
    assert (rc, out.strip()) == (0, "gh")


def test_LOC002_gitlab_host_is_glab(tmp_path):
    repo = make_repo(tmp_path, "https://gitlab.com/groupe/depot.git")
    rc, out = run(local_dir(tmp_path), "forge", "--repo", str(repo))
    assert (rc, out.strip()) == (0, "glab")


def test_LOC002_unknown_host_exits_1_and_is_named(tmp_path):
    repo = make_repo(tmp_path, "https://forge.exemple.test/groupe/depot.git")
    rc, out = run(local_dir(tmp_path), "forge", "--repo", str(repo))
    assert rc == 1
    assert "hôte forge.exemple.test inconnu" in out


def test_LOC003_real_case_line_naming_the_id_is_printed(tmp_path):
    line = "| PRV-015 | dépôt interne, critère X-103 |"
    local = local_dir(tmp_path, {"cas-reels.md": "| PRV-014 | autre |\n" + line + "\n"})
    rc, out = run(local, "cas-reels", "PRV-015")
    assert rc == 0
    assert line in out.splitlines()


def test_LOC003_missing_file_gives_empty_output(tmp_path):
    rc, out = run(local_dir(tmp_path), "cas-reels", "PRV-015")
    assert (rc, out) == (0, "")
