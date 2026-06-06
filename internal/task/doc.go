// Package task contains task executors and their registry.
//
// Design rules:
//  1. Service layer owns CRUD, scheduling, and execution log orchestration.
//  2. This package owns invoke-target specific validation and execution logic.
//  3. Every executor must provide a stable InvokeTarget, input validation, and
//     a deterministic execution result string for task logs and diagnostics.
//  4. New task types should be added by implementing Executor and registering
//     it in NewDefaultRegistry, instead of extending service-layer switch logic.
//  5. Example or experimental executors may live here without being registered
//     into the default registry until their contract is ready for production use.
package task
