# TFX – Multipath API Philosophy

> "If there's only one way to do it, it's not the right way."  
> — Core Principle of TFX

---

## 1. 🤔 Why Multipath?

Go encourages explicitness, but often ends up verbose — nudging library authors toward rigid **single-path APIs**: one constructor, one pattern, zero freedom.

**TFX** embraces a multi-entry design, offering **three coherent entry points** for every public-facing feature:

| Entry                 | Goal                                   | Typical Consumer               |
| --------------------- | -------------------------------------- | ------------------------------ |
| **Default / Express** | Zero-config, one-liner utility         | Scripts, demos                 |
| **Instantiated**      | Fine-grained control via config struct | Library authors, tool builders |
| **Fluent / DSL**      | Declarative chaining with readability  | Power users, prototyping       |

All styles share the same internal engine — no duplicated logic, no hidden side effects.

---

## 2. 🧰 Internal Consistency via `internal/share`

TFX provides the `Overload[T]` primitive to enable clean multipath APIs:

### 2.1 `Overload[T]`

```go
cfg := share.Overload[Config](args, DefaultConfig())
```

- Accepts **0 or 1** anonymous value.
- Handles both `Config` and `*Config`.
- Falls back safely to the provided default.
- Zero reflection — 100% compile-time safety.
- Uses Go generics for reusability.
- **Panics on invalid arguments** — ensures type safety at runtime.

**🚨 Enforcement**: All TFX packages **MUST** use `share.Overload[T]` for `...any` parameters. This maintains consistency across the ecosystem and provides predictable error behavior.

### 2.2 Real Implementation Example

Current TFX packages that follow this pattern include `progrefx` and `runfx`:

```go
// Primary function accepts ...any but uses Overload internally
func Start(opts ...any) *Progress {
	cfg := share.OverloadWithOptions(opts, DefaultProgressConfig())
	return newProgress(cfg)
}

// Usage examples:
progrefx.Start()                                          // Express: zero-config
progrefx.Start(progrefx.ProgressConfig{Total: 100})      // Instantiated: config struct
progrefx.NewProgressBuilder().Label("Sync").Build()      // DSL: builder + Build()
```

The `Start` function:

- Accepts `...any` for flexibility
- Uses `share.Overload` internally for type safety
- Supports both zero-config and config struct usage
- Panics if invalid arguments are provided

---

## 3. 📐 Canonical Pattern per Package

Every multipath package in TFX follows this baseline:

```go
// 1. Quick default (Express)
progrefx.Start()

// 2. Config struct (Instantiated)
cfg := progrefx.ProgressConfig{Total: 100, Label: "Sync"}
progrefx.Start(cfg)

// 3. Fluent builder (DSL)
progrefx.NewProgressBuilder().
    Total(100).
    Label("Sync").
    Build()

// 4. Full object lifecycle
bar := progrefx.NewProgressBuilder().Total(100).Build()
bar.Set(75)
bar.Finish()
```

### ✅ Enforcement Checklist

- Primary function (e.g., `Start()`, `Init()`, `Run()`) must work with **zero args**.
- Primary function must accept `...any` and use `share.Overload[T]` or `share.OverloadWithOptions[T]` internally.
- An instantiated form (`Start(cfg)` or similar) must exist.
- `New()` must exist for DSL chaining.
- All DSL methods must live in the **same package** and return the builder type.
- Builder must provide `.Build()` or an equivalent action method.
- If `...any` is exposed, provide clear documentation for expected types and panic behavior.

---

## 4. 🎯 Design Rationale

1. **DX First** — Libraries are judged by the first 30 seconds.
2. **No Reflection, No Magic** — Compile-time safety over runtime tricks.
3. **Opt-in Complexity** — Keep simple things simple. Expose power without requiring it.
4. **Uniform Mental Model** — Learn one, use everywhere.

---

## 5. 🧪 How to Build a New TFX Package

1. Define a `Config` struct with sensible defaults.
2. Provide `DefaultConfig()` function.
3. Create a builder type (e.g., `ProgressBuilder`) with chaining methods:

   ```go
   func (b *Builder) Label(text string) *Builder {
       b.config.Label = text
       return b
   }
   ```

4. Implement the primary function with `...any` and `share.Overload`:

   ```go
   func Start(args ...any) *YourType {
       cfg := share.Overload(args, DefaultConfig())
       return newYourType(cfg)
   }
   ```

5. Expose all three entry points: primary function (e.g., `Start()`), `Start(cfg)`, and `New()`.
6. Document all usage styles via GoDoc.

---

## 6. ❓ FAQ

**Q:** Isn't `...any` risky?  
**A:** It's scoped and type-switched with `share.Overload`. Regular users stay in safe, typed APIs.

**Q:** Why not use functional options?  
**A:** Builder patterns provide better IDE support and are more explicit about available options.

**Q:** What happens if I pass wrong arguments to `Start()`?  
**A:** `share.Overload` will panic with a clear error message. This is intentional — fail fast and loud.

**Q:** Isn't this over-engineered?  
**A:** Not if it lets beginners and power users coexist without friction.

---

## 🧠 Summary

The **Multipath API** is not a gimmick — it's a **DX multiplier**.  
It lets TFX be:

- Beginner-friendly in a one-liner.
- Scalable for complex CLI tools.
- Consistent and safe by design.

> Build APIs that scale **horizontally**, not just vertically.
