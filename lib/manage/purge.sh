#!/bin/bash
# Purge Module - Remove orphaned application files
# Finds and removes leftover files from previously uninstalled applications

set -euo pipefail

# Get script directory and source common functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
[[ -z "${MOLE_COMMON_LOADED:-}" ]] && source "$SCRIPT_DIR/lib/core/common.sh"

# ============================================================================
# Orphan Detection
# ============================================================================

# Common locations for application data
readonly PURGE_LOCATIONS=(
    "$HOME/Library/Application Support"
    "$HOME/Library/Caches"
    "$HOME/Library/Preferences"
    "$HOME/Library/Logs"
    "$HOME/Library/Containers"
    "$HOME/Library/Group Containers"
    "$HOME/Library/LaunchAgents"
    "$HOME/Library/Saved Application State"
)

# Get list of currently installed apps (bundle IDs and names)
get_installed_apps() {
    local -a apps=()

    # Scan /Applications
    for app in /Applications/*.app; do
        [[ -d "$app" ]] || continue
        local bundle_id=""
        if [[ -e "$app/Contents/Info.plist" ]]; then
            bundle_id=$(defaults read "$app/Contents/Info.plist" CFBundleIdentifier 2> /dev/null || echo "")
        fi
        local name=$(basename "$app" .app)
        [[ -n "$bundle_id" ]] && apps+=("$bundle_id")
        apps+=("$name")
    done

    # Scan ~/Applications
    if [[ -d "$HOME/Applications" ]]; then
        for app in "$HOME/Applications"/*.app; do
            [[ -d "$app" ]] || continue
            local bundle_id=""
            if [[ -e "$app/Contents/Info.plist" ]]; then
                bundle_id=$(defaults read "$app/Contents/Info.plist" CFBundleIdentifier 2> /dev/null || echo "")
            fi
            local name=$(basename "$app" .app)
            [[ -n "$bundle_id" ]] && apps+=("$bundle_id")
            apps+=("$name")
        done
    fi

    printf '%s\n' "${apps[@]}" | sort -u
}

# Check if a directory name matches an installed app
is_orphan_candidate() {
    local dir_name="$1"
    local installed_apps="$2"

    # Skip system directories
    case "$dir_name" in
        com.apple.* | Apple* | .* | System* | Utilities*)
            return 1
            ;;
    esac

    # Check if name matches any installed app
    if echo "$installed_apps" | grep -qi "^${dir_name}$"; then
        return 1 # Not an orphan
    fi

    # Check partial bundle ID match
    local base_name="${dir_name%%.*}"
    if echo "$installed_apps" | grep -qi "${base_name}"; then
        return 1 # Likely related to installed app
    fi

    return 0 # Potential orphan
}

# Find orphaned files
find_orphans() {
    local installed_apps
    installed_apps=$(get_installed_apps)

    local -a orphans=()
    local total_size=0

    for location in "${PURGE_LOCATIONS[@]}"; do
        [[ -d "$location" ]] || continue

        while IFS= read -r -d '' dir; do
            local name=$(basename "$dir")

            if is_orphan_candidate "$name" "$installed_apps"; then
                local size_kb=$(get_path_size_kb "$dir")
                orphans+=("$dir|$size_kb")
                ((total_size += size_kb))
            fi
        done < <(find "$location" -maxdepth 1 -type d ! -name "$(basename "$location")" -print0 2> /dev/null)
    done

    # Output orphans sorted by size (largest first)
    printf '%s\n' "${orphans[@]}" | sort -t'|' -k2 -nr

    # Return total size for summary
    echo "TOTAL|$total_size"
}

# ============================================================================
# CLI Interface
# ============================================================================

show_purge_help() {
    echo "Usage: mole purge [OPTIONS]"
    echo ""
    echo "Find and remove orphaned application files."
    echo ""
    echo "Options:"
    echo "  --scan          Scan for orphaned files (default)"
    echo "  --clean         Remove all detected orphans"
    echo "  --dry-run       Show what would be removed"
    echo "  --yes, -y       Skip confirmation prompt"
    echo "  --help, -h      Show this help message"
    echo ""
    echo "Examples:"
    echo "  mole purge --scan"
    echo "  mole purge --clean --yes"
}

# Parse flags
action="scan"
dry_run=false
auto_confirm=false

while [[ $# -gt 0 ]]; do
    case "$1" in
        --scan)
            action="scan"
            shift
            ;;
        --clean)
            action="clean"
            shift
            ;;
        --dry-run)
            dry_run=true
            shift
            ;;
        --yes | -y)
            auto_confirm=true
            shift
            ;;
        --help | -h)
            show_purge_help
            exit 0
            ;;
        *)
            log_warning "Unknown option: $1"
            shift
            ;;
    esac
done

# ============================================================================
# Main Execution
# ============================================================================

echo ""
echo -e "${PURPLE_BOLD}Orphan File Scanner${NC}"
echo ""

if [[ -t 1 ]]; then
    start_inline_spinner "Scanning for orphaned files..."
fi

# Get list of orphans
orphan_data=$(find_orphans)

if [[ -t 1 ]]; then
    stop_inline_spinner
fi

# Parse results
total_size=0
orphan_count=0
declare -a orphan_list=()

while IFS= read -r line; do
    [[ -z "$line" ]] && continue

    if [[ "$line" == TOTAL* ]]; then
        total_size="${line#TOTAL|}"
    else
        orphan_list+=("$line")
        ((orphan_count++))
    fi
done <<< "$orphan_data"

if [[ $orphan_count -eq 0 ]]; then
    echo -e "${GREEN}${ICON_SUCCESS}${NC} No orphaned files found"
    echo ""
    exit 0
fi

# Display results
size_display=$(bytes_to_human "$((total_size * 1024))")
echo -e "Found ${YELLOW}$orphan_count${NC} potential orphans (${size_display})"
echo ""

# Show top 10 largest
echo -e "${BLUE}Largest orphaned directories:${NC}"
count=0
for item in "${orphan_list[@]}"; do
    [[ $count -ge 10 ]] && break

    path="${item%|*}"
    size_kb="${item#*|}"
    size_human=$(bytes_to_human "$((size_kb * 1024))")

    echo -e "  ${GRAY}${size_human}${NC}  ${path/$HOME/~}"
    ((count++))
done

if [[ $orphan_count -gt 10 ]]; then
    echo -e "  ${GRAY}... and $((orphan_count - 10)) more${NC}"
fi
echo ""

# Handle clean action
if [[ "$action" == "clean" ]]; then
    if [[ "$dry_run" == "true" ]]; then
        echo -e "${YELLOW}[DRY RUN]${NC} Would remove $orphan_count orphaned directories"
        exit 0
    fi

    if [[ "$auto_confirm" != "true" ]]; then
        echo -ne "${PURPLE}${ICON_ARROW}${NC} Remove $orphan_count orphaned directories (${size_display})?  ${GREEN}Enter${NC} confirm, ${GRAY}ESC${NC} cancel: "
        IFS= read -r -s -n1 key || key=""
        case "$key" in
            $'\e' | q | Q)
                echo ""
                echo "Cancelled"
                exit 0
                ;;
            "" | $'\n' | $'\r' | y | Y)
                printf "\r\033[K"
                ;;
            *)
                echo ""
                echo "Cancelled"
                exit 0
                ;;
        esac
    fi

    # Remove orphans
    removed=0
    for item in "${orphan_list[@]}"; do
        path="${item%|*}"
        if safe_remove "$path" true; then
            ((removed++))
        fi
    done

    echo -e "${GREEN}${ICON_SUCCESS}${NC} Removed $removed orphaned directories, freed ${size_display}"
fi

echo ""
