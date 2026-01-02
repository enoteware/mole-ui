#!/bin/bash
# Batch Clean Module - Entry point for clean operations
# Handles CLI flags and delegates to appropriate cleanup functions

set -euo pipefail

# Get script directory and source common functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
[[ -z "${MOLE_COMMON_LOADED:-}" ]] && source "$SCRIPT_DIR/lib/core/common.sh"

# Source clean modules
source "$SCRIPT_DIR/lib/clean/caches.sh"
source "$SCRIPT_DIR/lib/clean/system.sh"
source "$SCRIPT_DIR/lib/clean/user.sh"
source "$SCRIPT_DIR/lib/clean/dev.sh"

# Parse command line arguments
DRY_RUN=false
YES_MODE=false
CLEAN_CACHE=false
CLEAN_LOGS=false
CLEAN_DOWNLOADS=false
CLEAN_XCODE=false
CLEAN_ALL=false
VERBOSE=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --dry-run|-n)
            DRY_RUN=true
            shift
            ;;
        --yes|-y)
            YES_MODE=true
            shift
            ;;
        --cache|--caches)
            CLEAN_CACHE=true
            shift
            ;;
        --logs)
            CLEAN_LOGS=true
            shift
            ;;
        --downloads)
            CLEAN_DOWNLOADS=true
            shift
            ;;
        --xcode)
            CLEAN_XCODE=true
            shift
            ;;
        --all)
            CLEAN_ALL=true
            shift
            ;;
        --verbose|-v)
            VERBOSE=true
            shift
            ;;
        --debug)
            DEBUG_MODE=true
            shift
            ;;
        *)
            shift
            ;;
    esac
done

# If no specific clean type is selected, default to all
if [[ "$CLEAN_CACHE" == "false" && "$CLEAN_LOGS" == "false" && "$CLEAN_DOWNLOADS" == "false" && "$CLEAN_XCODE" == "false" ]]; then
    CLEAN_ALL=true
fi

# Export flags for use by clean modules
export MOLE_DRY_RUN="$DRY_RUN"
export MOLE_YES_MODE="$YES_MODE"
export MOLE_VERBOSE="$VERBOSE"

# Initialize size tracking
TOTAL_FREED=0

# ============================================================================
# Clean Functions
# ============================================================================

clean_caches() {
    echo ""
    log_section "Cleaning Caches"

    local before_size=$(get_path_size_kb "$HOME/Library/Caches" 2>/dev/null || echo "0")

    # Clean user caches
    if [[ -d "$HOME/Library/Caches" ]]; then
        if [[ "$DRY_RUN" == "true" ]]; then
            echo "Would clean: ~/Library/Caches"
        else
            # Safe cache cleaning - only old files
            find "$HOME/Library/Caches" -type f -mtime +7 -delete 2>/dev/null || true
            log_success "User caches cleaned"
        fi
    fi

    # Clean browser caches
    local chrome_cache="$HOME/Library/Caches/Google/Chrome"
    local safari_cache="$HOME/Library/Caches/com.apple.Safari"

    for cache_dir in "$chrome_cache" "$safari_cache"; do
        if [[ -d "$cache_dir" ]]; then
            if [[ "$DRY_RUN" == "true" ]]; then
                echo "Would clean: $cache_dir"
            else
                find "$cache_dir" -type f -mtime +3 -delete 2>/dev/null || true
            fi
        fi
    done

    local after_size=$(get_path_size_kb "$HOME/Library/Caches" 2>/dev/null || echo "0")
    local freed=$((before_size - after_size))
    [[ $freed -gt 0 ]] && TOTAL_FREED=$((TOTAL_FREED + freed))

    log_success "Browser caches cleaned"
}

clean_logs() {
    echo ""
    log_section "Cleaning Logs"

    local log_dirs=(
        "$HOME/Library/Logs"
        "$HOME/Library/Application Support/CrashReporter"
    )

    for log_dir in "${log_dirs[@]}"; do
        if [[ -d "$log_dir" ]]; then
            local before_size=$(get_path_size_kb "$log_dir" 2>/dev/null || echo "0")

            if [[ "$DRY_RUN" == "true" ]]; then
                echo "Would clean: $log_dir"
            else
                find "$log_dir" -type f \( -name "*.log" -o -name "*.crash" \) -mtime +7 -delete 2>/dev/null || true
            fi

            local after_size=$(get_path_size_kb "$log_dir" 2>/dev/null || echo "0")
            local freed=$((before_size - after_size))
            [[ $freed -gt 0 ]] && TOTAL_FREED=$((TOTAL_FREED + freed))
        fi
    done

    log_success "Log files cleaned"
}

clean_downloads() {
    echo ""
    log_section "Cleaning Downloads"

    local downloads_dir="$HOME/Downloads"

    if [[ -d "$downloads_dir" ]]; then
        local before_size=$(get_path_size_kb "$downloads_dir" 2>/dev/null || echo "0")

        # Only clean old DMG, ZIP, and installer files
        local patterns=("*.dmg" "*.pkg" "*.zip" "*.tar.gz" "*.tgz")

        for pattern in "${patterns[@]}"; do
            if [[ "$DRY_RUN" == "true" ]]; then
                find "$downloads_dir" -maxdepth 1 -name "$pattern" -mtime +30 -print 2>/dev/null || true
            else
                find "$downloads_dir" -maxdepth 1 -name "$pattern" -mtime +30 -delete 2>/dev/null || true
            fi
        done

        local after_size=$(get_path_size_kb "$downloads_dir" 2>/dev/null || echo "0")
        local freed=$((before_size - after_size))
        [[ $freed -gt 0 ]] && TOTAL_FREED=$((TOTAL_FREED + freed))

        log_success "Old downloads cleaned (DMG, ZIP, PKG files older than 30 days)"
    fi
}

clean_xcode() {
    echo ""
    log_section "Cleaning Xcode"

    local xcode_dirs=(
        "$HOME/Library/Developer/Xcode/DerivedData"
        "$HOME/Library/Developer/Xcode/Archives"
        "$HOME/Library/Developer/CoreSimulator/Caches"
    )

    for xcode_dir in "${xcode_dirs[@]}"; do
        if [[ -d "$xcode_dir" ]]; then
            local size=$(get_path_size_kb "$xcode_dir" 2>/dev/null || echo "0")

            if [[ "$DRY_RUN" == "true" ]]; then
                echo "Would clean: $xcode_dir ($(format_size_kb $size))"
            else
                rm -rf "$xcode_dir"/* 2>/dev/null || true
                TOTAL_FREED=$((TOTAL_FREED + size))
                log_success "Cleaned $(basename "$xcode_dir")"
            fi
        fi
    done
}

# ============================================================================
# Main Execution
# ============================================================================

echo ""
echo -e "${PURPLE}${ICON_MOLE}${NC} ${BOLD}Mole Cleaner${NC}"
echo ""

if [[ "$DRY_RUN" == "true" ]]; then
    echo -e "${YELLOW}Dry run mode - no files will be deleted${NC}"
    echo ""
fi

# Execute requested clean operations
if [[ "$CLEAN_ALL" == "true" ]] || [[ "$CLEAN_CACHE" == "true" ]]; then
    clean_caches
fi

if [[ "$CLEAN_ALL" == "true" ]] || [[ "$CLEAN_LOGS" == "true" ]]; then
    clean_logs
fi

if [[ "$CLEAN_DOWNLOADS" == "true" ]]; then
    clean_downloads
fi

if [[ "$CLEAN_XCODE" == "true" ]]; then
    clean_xcode
fi

# Summary
echo ""
echo "======================================================================"
if [[ "$DRY_RUN" == "true" ]]; then
    echo "Dry run complete - no files were deleted"
else
    echo -e "Clean complete! Freed: ${GREEN}$(format_size_kb $TOTAL_FREED)${NC}"
fi
echo "======================================================================"
