---
name: git_safe_merge
description: Executes a safe git merge workflow with automatic backup, validation steps, and rollback capabilities.
---

# Git Safe Merge Skill

This skill executes a robust merge workflow designed to handle upstream updates safely. It includes branching for safety, merging, conflict resolution guidance, and validation hooks.

## 1. Preparation & Backup

Before merging, ensure the working directory is clean and create a safety backup.

1.  **Check Status**:
    ```bash
    git status
    ```
    *Stop if not clean.*

2.  **Create Backup Branch**:
    Creates a timestamped backup of the current state.
    ```bash
    CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
    BACKUP_BRANCH="backup/${CURRENT_BRANCH}_$(date +%Y%m%d_%H%M%S)"
    git branch "$BACKUP_BRANCH"
    echo "✅ Backup created: $BACKUP_BRANCH"
    ```

## 2. Merge Execution

Perform the merge from the specified source (default: `upstream/main`).

1.  **Fetch Latest**:
    ```bash
    git fetch upstream
    ```

2.  **Execute Merge**:
    ```bash
    git merge upstream/main
    ```

3.  **Conflict Check**:
    Check if the merge resulted in conflicts.
    ```bash
    if [ $? -ne 0 ]; then
        echo "⚠️ Merge conflicts detected."
        echo "List of conflicting files:"
        git diff --name-only --diff-filter=U
        exit 1
    else
        echo "✅ Merge successful (auto-merged)."
    fi
    ```

## 3. Conflict Resolution & Validation

If conflicts occur:

1.  **Identify Conflicts**:
    The command `git status` will list files in conflict.

2.  **Manual Resolution**:
    Open each conflicting file, look for `<<<<<<<`, `=======`, `>>>>>>>` markers, and resolve the code.

3.  **Continue Merge**:
    After resolving conflicts and staging files (`git add .`):
    ```bash
    git commit -m "Merge upstream changes and resolve conflicts"
    ```

4.  **Verification**:
    Run project build or test commands to ensure stability.
    ```bash
    # Example
    # make build
    # make test
    ```

## 4. Rollback (If Needed)

If the merge creates critical issues that cannot be easily fixed, restore from the backup.

```bash
# To be run manually by user if needed
git reset --hard "$BACKUP_BRANCH"
echo "🔄 Rolled back to $BACKUP_BRANCH"
```
