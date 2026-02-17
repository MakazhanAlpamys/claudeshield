#!/bin/bash
# ClaudeShield Policy Shell Wrapper
# This script acts as a proxy shell that checks commands against policy rules
# before executing them. It reads rules from /etc/claudeshield/policy.json.

POLICY_FILE="/etc/claudeshield/policy.json"
AUDIT_LOG="/workspace/.claudeshield/shell-audit.jsonl"

# Ensure audit dir exists
mkdir -p "$(dirname "$AUDIT_LOG")" 2>/dev/null

log_entry() {
    local action="$1" command="$2" reason="$3"
    local ts
    ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    printf '{"timestamp":"%s","event_type":"shell_exec","command":"%s","action":"%s","reason":"%s"}\n' \
        "$ts" "$(echo "$command" | sed 's/"/\\"/g')" "$action" "$(echo "$reason" | sed 's/"/\\"/g')" \
        >> "$AUDIT_LOG" 2>/dev/null
}

check_policy() {
    local cmd="$1"

    # If no policy file, allow everything (fail-open when no policy configured)
    if [ ! -f "$POLICY_FILE" ]; then
        return 0
    fi

    # Check block rules first (deny takes priority)
    local block_count
    block_count=$(jq -r '.block | length' "$POLICY_FILE" 2>/dev/null)
    if [ -n "$block_count" ] && [ "$block_count" -gt 0 ]; then
        local i=0
        while [ "$i" -lt "$block_count" ]; do
            local pattern reason
            pattern=$(jq -r ".block[$i].pattern" "$POLICY_FILE")
            reason=$(jq -r ".block[$i].reason // \"Blocked by policy\"" "$POLICY_FILE")

            # Convert glob pattern to a check
            if match_pattern "$pattern" "$cmd"; then
                log_entry "block" "$cmd" "$reason"
                echo "ClaudeShield: BLOCKED — $reason" >&2
                return 1
            fi
            i=$((i + 1))
        done
    fi

    # Check allow rules
    local allow_count
    allow_count=$(jq -r '.allow | length' "$POLICY_FILE" 2>/dev/null)
    if [ -n "$allow_count" ] && [ "$allow_count" -gt 0 ]; then
        local i=0
        while [ "$i" -lt "$allow_count" ]; do
            local pattern
            pattern=$(jq -r ".allow[$i].pattern" "$POLICY_FILE")

            if match_pattern "$pattern" "$cmd"; then
                log_entry "allow" "$cmd" ""
                return 0
            fi
            i=$((i + 1))
        done
    fi

    # Default: block (fail-secure)
    log_entry "block" "$cmd" "Command not in allowlist"
    echo "ClaudeShield: BLOCKED — Command not in allowlist" >&2
    return 1
}

match_pattern() {
    local pattern="$1" cmd="$2"

    # Handle "command *" pattern — matches "command" or "command anything"
    if [[ "$pattern" == *" *" ]]; then
        local prefix="${pattern% \*}"
        if [ "$cmd" = "$prefix" ] || [[ "$cmd" == "$prefix "* ]]; then
            return 0
        fi
        return 1
    fi

    # Handle "*suffix" pattern
    if [[ "$pattern" == \** ]]; then
        local suffix="${pattern#\*}"
        if [[ "$cmd" == *"$suffix" ]]; then
            return 0
        fi
        return 1
    fi

    # Exact match
    if [ "$cmd" = "$pattern" ]; then
        return 0
    fi

    # Glob match via case
    case "$cmd" in
        $pattern) return 0 ;;
    esac

    return 1
}

# If called with -c (standard shell exec mode), check the command
if [ "$1" = "-c" ]; then
    shift
    FULL_CMD="$*"

    # Extract the base command (first word/pipeline segment)
    check_policy "$FULL_CMD"
    if [ $? -ne 0 ]; then
        exit 126
    fi

    exec /bin/bash -c "$FULL_CMD"
else
    # Interactive mode — pass through to bash
    exec /bin/bash "$@"
fi
