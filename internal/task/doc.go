// Package task contains task-domain documentation only.
//
// Design rules:
//  1. Service layer owns CRUD and coordinates scheduling/runtime injection.
//  2. Package runtime contains generic runtime contracts and scheduler/runner.
//  3. Package tasks/<name> owns one concrete task's contract, factory,
//     executor, and registration.
//  4. The root task package owns task-domain assembly helpers such as loader
//     and store adapters.
package task
