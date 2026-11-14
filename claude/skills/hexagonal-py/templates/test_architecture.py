"""Architecture criteria ARC-001 to ARC-003.

Reads the import graph of the package; imports nothing of it.
"""

import copy
import sys

import grimp
import pytest

PACKAGE = "<pkg>"
DOMAIN = f"{PACKAGE}.domain"
APP = f"{PACKAGE}.app"
ADAPTERS = f"{PACKAGE}.adapters"
ROOT = f"{PACKAGE}.composition"

# The third-party packages each layer may import. Adding one is a criterion
# change: ARC-001 for the domain, ARC-002 for the use cases.
ALLOWED_DOMAIN: frozenset[str] = frozenset()
ALLOWED_APP: frozenset[str] = frozenset()


@pytest.fixture(scope="module")
def graph() -> grimp.ImportGraph:
    return grimp.build_graph(PACKAGE, include_external_packages=True)


def under(module: str, package: str) -> bool:
    return module == package or module.startswith(package + ".")


def listed(module: str, allowed: frozenset[str]) -> bool:
    top = module.split(".")[0]
    return top in sys.stdlib_module_names or top in allowed


def members(graph: grimp.ImportGraph, package: str) -> set[str]:
    found = {m for m in graph.modules if under(m, package)}
    assert found, f"no module under {package}"
    return found


def chain(graph: grimp.ImportGraph, importer: str, imported: str) -> str:
    found = graph.find_shortest_chain(importer=importer, imported=imported)
    return " -> ".join(found or (importer, imported))


def test_ARC001_domain_depends_only_on_itself(graph: grimp.ImportGraph) -> None:
    offending = {
        chain(graph, module, dep)
        for module in members(graph, DOMAIN)
        for dep in graph.find_upstream_modules(module)
        if not (under(dep, DOMAIN) or listed(dep, ALLOWED_DOMAIN))
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
        if not (under(dep, APP) or listed(dep, ALLOWED_APP))
    }
    assert not offending, f"a use case depends on: {sorted(offending)}"


def test_ARC003_only_the_composition_root_assembles(graph: grimp.ImportGraph) -> None:
    def adapter(module: str) -> str:
        inside = under(module, ADAPTERS) and module != ADAPTERS
        return module.split(".")[2] if inside else ""

    def judged_elsewhere(importer: str) -> bool:
        # The domain by ARC-001, the use cases by ARC-002; the root may assemble.
        return under(importer, DOMAIN) or under(importer, APP) or importer == ROOT

    def inside_adapters(importer: str, module: str) -> bool:
        # The package may re-export its adapters; an adapter may import itself.
        own = adapter(module) != "" and adapter(importer) == adapter(module)
        return importer == ADAPTERS or own

    offending = {
        f"{importer} -> {module}"
        for module in members(graph, ADAPTERS)
        for importer in graph.find_modules_that_directly_import(module)
        if not (judged_elsewhere(importer) or inside_adapters(importer, module))
    }
    assert not offending, f"only {ROOT} may import an adapter: {sorted(offending)}"
