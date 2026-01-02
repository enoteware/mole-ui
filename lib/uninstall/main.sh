#!/bin/bash
# Interactive Uninstall Module - Terminal-based app selection UI
# Provides an interactive menu for selecting and uninstalling applications

set -euo pipefail

# Get script directory and source common functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
[[ -z "${MOLE_COMMON_LOADED:-}" ]] && source "$SCRIPT_DIR/lib/core/common.sh"
source "$SCRIPT_DIR/lib/uninstall/batch.sh"

# ============================================================================
# Application Scanning
# ============================================================================

# Scan for installed applications
scan_applications() {
    local -a apps=()
    local idx=1

    # Scan /Applications
    while IFS= read -r -d '' app_path; do
        [[ -d "$app_path" ]] || continue

        # Skip protected apps
        if is_protected_app_path "$app_path"; then
            continue
        fi

        local app_name=$(basename "$app_path" .app)
        local bundle_id="unknown"
        local size_kb=0

        if [[ -e "$app_path/Contents/Info.plist" ]]; then
            bundle_id=$(defaults read "$app_path/Contents/Info.plist" CFBundleIdentifier 2>/dev/null || echo "unknown")
        fi

        size_kb=$(get_path_size_kb "$app_path")

        # Format: index|path|name|bundle_id|size_kb|extra
        apps+=("$idx|$app_path|$app_name|$bundle_id|$size_kb|")
        ((idx++))
    done < <(find /Applications -maxdepth 1 -name "*.app" -type d -print0 2>/dev/null | sort -z)

    # Scan ~/Applications if exists
    if [[ -d "$HOME/Applications" ]]; then
        while IFS= read -r -d '' app_path; do
            [[ -d "$app_path" ]] || continue

            local app_name=$(basename "$app_path" .app)
            local bundle_id="unknown"
            local size_kb=0

            if [[ -e "$app_path/Contents/Info.plist" ]]; then
                bundle_id=$(defaults read "$app_path/Contents/Info.plist" CFBundleIdentifier 2>/dev/null || echo "unknown")
            fi

            size_kb=$(get_path_size_kb "$app_path")

            apps+=("$idx|$app_path|$app_name|$bundle_id|$size_kb|")
            ((idx++))
        done < <(find "$HOME/Applications" -maxdepth 1 -name "*.app" -type d -print0 2>/dev/null | sort -z)
    fi

    printf '%s\n' "${apps[@]}"
}

# ============================================================================
# Interactive Menu
# ============================================================================

# Display the application selection menu
show_app_menu() {
    local -a apps=("$@")
    local -a selected=()
    local current=0
    local page=0
    local page_size=15
    local total=${#apps[@]}
    local total_pages=$(( (total + page_size - 1) / page_size ))

    # Initialize selected array (all false)
    for ((i=0; i<total; i++)); do
        selected[$i]=false
    done

    # Hide cursor
    tput civis 2>/dev/null || true
    trap 'tput cnorm 2>/dev/null || true' EXIT

    while true; do
        # Clear screen and show header
        clear
        echo -e "${PURPLE_BOLD}Select Applications to Uninstall${NC}"
        echo -e "${GRAY}Use arrow keys to navigate, Space to select, Enter to confirm${NC}"
        echo ""

        # Calculate page range
        local start=$((page * page_size))
        local end=$((start + page_size))
        [[ $end -gt $total ]] && end=$total

        # Display apps for current page
        for ((i=start; i<end; i++)); do
            local entry="${apps[$i]}"
            IFS='|' read -r idx app_path app_name bundle_id size_kb extra <<< "$entry"

            local size_human=$(bytes_to_human "$((size_kb * 1024))")
            local prefix="  "
            local checkbox="[ ]"

            # Highlight current item
            if [[ $i -eq $current ]]; then
                prefix="> "
            fi

            # Show selection state
            if [[ "${selected[$i]}" == "true" ]]; then
                checkbox="${GREEN}[x]${NC}"
            fi

            if [[ $i -eq $current ]]; then
                echo -e "${prefix}${checkbox} ${BLUE}${app_name}${NC} ${GRAY}(${size_human})${NC}"
            else
                echo -e "${prefix}${checkbox} ${app_name} ${GRAY}(${size_human})${NC}"
            fi
        done

        # Show page info and controls
        echo ""
        echo -e "${GRAY}Page $((page + 1))/$total_pages | q=quit, a=select all, n=select none${NC}"

        # Count selected
        local selected_count=0
        for s in "${selected[@]}"; do
            [[ "$s" == "true" ]] && ((selected_count++))
        done

        if [[ $selected_count -gt 0 ]]; then
            echo -e "${GREEN}$selected_count app(s) selected${NC} - Press Enter to uninstall"
        fi

        # Read key input
        IFS= read -r -s -n1 key

        case "$key" in
            $'\x1b')  # Escape sequence
                read -r -s -n2 -t 0.1 seq || true
                case "$seq" in
                    '[A')  # Up arrow
                        ((current > 0)) && ((current--))
                        # Adjust page if needed
                        if [[ $current -lt $((page * page_size)) ]]; then
                            ((page > 0)) && ((page--))
                        fi
                        ;;
                    '[B')  # Down arrow
                        ((current < total - 1)) && ((current++))
                        # Adjust page if needed
                        if [[ $current -ge $(((page + 1) * page_size)) ]]; then
                            ((page < total_pages - 1)) && ((page++))
                        fi
                        ;;
                    '[5')  # Page Up
                        read -r -s -n1 -t 0.1 _ || true
                        ((page > 0)) && ((page--))
                        current=$((page * page_size))
                        ;;
                    '[6')  # Page Down
                        read -r -s -n1 -t 0.1 _ || true
                        ((page < total_pages - 1)) && ((page++))
                        current=$((page * page_size))
                        ;;
                esac
                ;;
            ' ')  # Space - toggle selection
                if [[ "${selected[$current]}" == "true" ]]; then
                    selected[$current]=false
                else
                    selected[$current]=true
                fi
                ;;
            $'\n' | '')  # Enter - confirm
                if [[ $selected_count -gt 0 ]]; then
                    break
                fi
                ;;
            'a' | 'A')  # Select all
                for ((i=0; i<total; i++)); do
                    selected[$i]=true
                done
                ;;
            'n' | 'N')  # Select none
                for ((i=0; i<total; i++)); do
                    selected[$i]=false
                done
                ;;
            'q' | 'Q')  # Quit
                tput cnorm 2>/dev/null || true
                clear
                echo "Cancelled"
                exit 0
                ;;
        esac
    done

    # Show cursor again
    tput cnorm 2>/dev/null || true
    clear

    # Build selected_apps array for batch_uninstall_applications
    selected_apps=()
    for ((i=0; i<total; i++)); do
        if [[ "${selected[$i]}" == "true" ]]; then
            selected_apps+=("${apps[$i]}")
        fi
    done

    # Run batch uninstall
    if [[ ${#selected_apps[@]} -gt 0 ]]; then
        batch_uninstall_applications
    fi
}

# ============================================================================
# Main Entry Point
# ============================================================================

echo ""
echo -e "${PURPLE_BOLD}Application Uninstaller${NC}"
echo ""

if [[ -t 1 ]]; then
    start_inline_spinner "Scanning applications..."
fi

# Scan for apps
mapfile -t app_list < <(scan_applications)

if [[ -t 1 ]]; then
    stop_inline_spinner
fi

if [[ ${#app_list[@]} -eq 0 ]]; then
    echo -e "${YELLOW}No applications found${NC}"
    exit 0
fi

echo "Found ${#app_list[@]} applications"
echo ""

# Show interactive menu
show_app_menu "${app_list[@]}"
