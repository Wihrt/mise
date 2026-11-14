"""Architecture criteria ARC-001 to ARC-003.

Reads the import graph of the package; imports nothing of it.
"""

import copy
import sys

import grimp
import pytest

PACKAGE = "<pkg>"  # the package under src/
DOMAIN = f"{PACKAGE}.domain"
APP = f"{PACKAGE}.app"
ADAPTERS = f"{PACKAGE}.adapters"
ENTRYPOINTS = f"{PACKAGE}.entrypoints"
ROOT = f"{PACKAGE}.composition"

# Third-party packages the domain may import. Adding one is a criterion change of ARC-001.
ALLOWED: frozenset[str] = frozenset()


@pytest.fixture(scope="module")
def graph() -> grimp.ImportGraph:
    return grimp.build_graph(PACKAGE, include_external_packages=True)


def under(module: str, package: str) -> bool:
    return module == package or module.startswith(package + ".")


def members(graph: grimp.ImportGraph, package: str) -> set[str]:
    found = {m for m in graph.modules if under(m, package)}
    assert found, f"no module under {package}"
    return found


def chain(graph: grimp.ImportGraph, importer: str, imported: str) -> str:
    return " -> ".join(graph.find_shortest_chain(importer=importer, imported=imported) or (importer, imported))


def test_ARC001_domain_depends_only_on_itself(graph: grimp.ImportGraph) -> None:
    def permitted(module: str) -> bool:
        top = module.split(".")[0]
        return under(module, DOMAIN) or top in sys.stdlib_module_names or top in ALLOWED

    offending = {
        chain(graph, module, dep)
        for module in members(graph, DOMAIN)
        for dep in graph.find_upstream_modules(module)
        if not permitted(dep)
    }
    assert not offending, f"the domain depends on: {sorted(offending)}"


def test_ARC002_use_cases_know_no_technology(graph: grimp.ImportGraph) -> None:
    # What the domain imports is judged by ARC-001, not here.
    outside_domain = copy.deepcopy(graph)
    for module in members(graph, DOMAIN):
        outside_domain.remove_module(module)
    offending = {
        chain(outside_domain, module, dep)
        for module in members(outside_domain, APP)
        for dep in outside_domain.find_upstream_modules(module)
        if under(dep, ADAPTERS) or under(dep, ENTRYPOINTS) or dep == ROOT
    }
    assert not offending, f"a use case depends on: {sorted(offending)}"


def test_ARC003_only_the_composition_root_assembles(graph: grimp.ImportGraph) -> None:
    def adapter(module: str) -> str:
        return module.split(".")[2] if under(module, ADAPTERS) and module != ADAPTERS else ""

    offending = {
        f"{importer} -> {module}"
        for module in members(graph, ADAPTERS)
        if adapter(module)
        for importer in graph.find_modules_that_directly_import(module)
        # The domain and the use cases are judged by ARC-001 and ARC-002; the root may assemble.
        if not (under(importer, DOMAIN) or under(importer, APP) or importer == ROOT or adapter(importer) == adapter(module))
    }
    assert not offending, f"only {ROOT} may import an adapter: {sorted(offending)}"
