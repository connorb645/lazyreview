package testsupport

import "testing"

// WorkbenchPulls are the workbench's pull requests:
//
//	#21  feature/search on main         adds Search, edits MemoryCatalog
//	#22  feature/basket on #21's branch  edits Basket and cart.ts
//	#30  fix/wording on main             edits docs/guide.md
var WorkbenchPulls = []WorkbenchPull{
	{Number: 21, Base: "main", Head: "feature/search", Title: "Add search to the catalog", Author: "sam", Body: "Adds a search over the catalog by name."},
	{Number: 22, Base: "feature/search", Head: "feature/basket", Title: "Add a basket that uses search", Author: "sam"},
	{Number: 30, Base: "main", Head: "fix/wording", Title: "Fix the guide wording", Author: "alex"},
}

// NewWorkbench makes the workbench: a repository holding a stack of two
// pull requests and a single one, as WorkbenchPulls names them, with main
// checked out. Its commits are made at the fixed time, so they are the same
// every run.
func NewWorkbench(t *testing.T) *ThrowawayRepo {
	t.Helper()
	r := NewThrowawayRepo(t)
	r.Commit("main", map[string]string{
		"Sources/Catalog/Catalog.swift":       "protocol Catalog {\n    func items() -> [Item]\n}\n\nstruct Item {\n    let name: String\n}\n",
		"Sources/Catalog/MemoryCatalog.swift": "final class MemoryCatalog: Catalog {\n    private var stored: [Item] = []\n\n    func items() -> [Item] {\n        stored\n    }\n}\n",
		"Sources/Basket/Basket.swift":         "final class Basket {\n    let catalog: Catalog\n\n    init(catalog: Catalog) {\n        self.catalog = catalog\n    }\n\n    func total() -> Int {\n        0\n    }\n}\n",
		"web/src/cart.ts":                     "export function cartTotal(prices: number[]): number {\n  return prices.reduce((a, b) => a + b, 0);\n}\n",
		"docs/guide.md":                       "# Guide\n\nThe catalog lists items.\n\n- Browse\n- Search\n",
	}, "base")
	r.Commit("feature/search", map[string]string{
		"Sources/Catalog/Search.swift":        "struct Search {\n    let catalog: Catalog\n\n    func find(_ text: String) -> [Item] {\n        catalog.items().filter { $0.name.contains(text) }\n    }\n}\n",
		"Sources/Catalog/MemoryCatalog.swift": "final class MemoryCatalog: Catalog {\n    private var stored: [Item] = []\n\n    func items() -> [Item] {\n        stored\n    }\n\n    func add(_ item: Item) {\n        stored.append(item)\n    }\n}\n",
	}, "Add search")
	r.Commit("feature/basket", map[string]string{
		"Sources/Basket/Basket.swift": "final class Basket {\n    let catalog: Catalog\n    let search: Search\n\n    init(catalog: Catalog) {\n        self.catalog = catalog\n        self.search = Search(catalog: catalog)\n    }\n\n    func total() -> Int {\n        1\n    }\n}\n",
		"web/src/cart.ts":             "export function cartTotal(prices: number[]): number {\n  if (!prices.length) return 0;\n  return prices.reduce((a, b) => a + b, 0);\n}\n",
	}, "Add basket")
	r.Checkout("main")
	r.Commit("fix/wording", map[string]string{"docs/guide.md": "# Guide\n\nThe catalog lists every item.\n\n- Browse\n- Search\n"}, "Fix wording")
	r.Checkout("main")
	return r
}
