# Yeast MVP Development Progress

## Phase 1: Foundation & Configuration
- [x] **Task 1: Project Initialization & Config Parsing**
  - **Goal**: Set up the Go project structure and implement the ability to read and parse the `yeast.yaml` configuration file.
  - **Details**:
    - Initialize Go module (`go mod init yeast`).
    - Create standard folder structure (`cmd/yeast`, `internal/config`, `internal/types`).
    - Define the Go structs for `yeast.yaml` based on the MVP spec.
    - Implement the YAML parser.
  - **Testing**: Write unit tests to ensure valid configs are parsed correctly and invalid configs return helpful errors.

## Phase 2: Core Virtualization Logic (QEMU Wrappers)
- [x] **Task 2: QEMU Command Construction**
  - **Goal**: Create a specific Go package to generate the exact CLI arguments for `qemu-system-x86_64`.
  - **Details**:
    - Create `internal/qemu`.
    - Implement functions to build flags for: KVM enable, memory, CPU, drives (virtio), networking (user mode + port forward), and display (nographic).
  - **Testing**: Unit tests that take a config struct and assert the returned slice of strings (command args) is exactly as expected.

## Phase 3: Cloud-Init Data Generation
- [x] **Task 3: Cloud-Init ISO Generation**
  - **Goal**: Generate `user-data` and `meta-data` files and pack them into a seed ISO.
  - **Details**:
    - Create `internal/cloudinit`.
    - Define templates for `user-data` (users, packages, runcmd).
    - Implement a function to write these to a temporary dir and use `genisoimage` (or `xorriso`) to build `seed.iso`.
  - **Testing**: Verify that the generated text content matches the template and that the ISO creation command is correctly formulated.

## Phase 4: Workspace & Image Management
- [ ] **Task 4: Workspace & Image Validation**
  - **Goal**: Manage the `.yeast` local directory and validate base images.
  - **Details**:
    - Create `internal/workspace`.
    - Logic to check/create `.yeast` directory.
    - Logic to verify if the specified base image (e.g., `ubuntu-22.04.img`) exists in the project root or image path.
  - **Testing**: Tests with temporary directories and dummy files to verify existence checks and directory creation.

## Phase 5: VM Lifecycle (The "Up" Command)
- [ ] **Task 5: VM Instantiation ("Up" Logic)**
  - **Goal**: Tie Config, QEMU, Cloud-Init, and Workspace together to actually start a process.
  - **Details**:
    - Create `internal/manager`.
    - Implement `Up()`: Create overlay image, generate seed ISO, execute QEMU command in background, write state to `.yeast/state.json`.
  - **Testing**: Integration test (mocking the actual `exec.Command`) to verify the sequence of operations.

## Phase 6: VM Lifecycle (Status & Down)
- [ ] **Task 6: State Management ("Status" & "Down")**
  - **Goal**: Read state to show status and stop the VM.
  - **Details**:
    - Implement `Status()`: Check if PID from `state.json` is alive.
    - Implement `Down()`: Send SIGTERM/SIGKILL to PID, cleanup `state.json`.
  - **Testing**: Test with dummy processes to ensure PID handling works.

## Phase 7: SSH Access
- [ ] **Task 7: SSH Connection**
  - **Goal**: Allow user to connect to the running VM.
  - **Details**:
    - Implement `SSH()`: Read port from state, exec `ssh -p <port> user@localhost`.
  - **Testing**: Verify command construction.

## Phase 8: CLI UX Entrypoint
- [ ] **Task 8: CLI Wiring (Cobra)**
  - **Goal**: Create the user-facing CLI.
  - **Details**:
    - Use `cobra` to create `main.go` commands: `yeast up`, `yeast down`, `yeast status`, `yeast ssh`.
    - Wire these commands to the `internal/manager` functions.
    - Implement "Calm UX" output (custom logger/printer).
  - **Testing**: Run the binary and verify help output and basic flow.
