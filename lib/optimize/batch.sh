#!/bin/bash
# Batch Optimize Module - Entry point for system optimization operations
# Runs maintenance tasks, cache refresh, and system cleanup

set -euo pipefail

# Get script directory and source common functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
[[ -z "${MOLE_COMMON_LOADED:-}" ]] && source "$SCRIPT_DIR/lib/core/common.sh"

# Source optimization modules
source "$SCRIPT_DIR/lib/optimize/maintenance.sh"
source "$SCRIPT_DIR/lib/optimize/tasks.sh"

# ============================================================================
# CLI Flag Parsing
# ============================================================================

show_optimize_help() {
    echo "Usage: mole optimize [OPTIONS]"
    echo ""
    echo "System optimization and maintenance tasks."
    echo ""
    echo "Options:"
    echo "  --dns           Flush DNS cache"
    echo "  --caches        Refresh system caches (QuickLook, icons, Safari)"
    echo "  --network       Full network optimization (DNS, ARP, mDNS)"
    echo "  --maintenance   Run system maintenance scripts"
    echo "  --snapshots     Thin Time Machine local snapshots"
    echo "  --prefs         Fix broken preference files"
    echo "  --login         Fix broken login items"
    echo "  --all           Run all optimization tasks"
    echo "  --dry-run       Show what would be done without making changes"
    echo "  --help, -h      Show this help message"
    echo ""
    echo "Examples:"
    echo "  mole optimize --all"
    echo "  mole optimize --dns --caches"
    echo "  mole optimize --network --maintenance"
}

# Parse command line flags
run_dns=false
run_caches=false
run_network=false
run_maintenance=false
run_snapshots=false
run_prefs=false
run_login=false
run_all=false
dry_run=false

while [[ $# -gt 0 ]]; do
    case "$1" in
        --dns)
            run_dns=true
            shift
            ;;
        --caches)
            run_caches=true
            shift
            ;;
        --network)
            run_network=true
            shift
            ;;
        --maintenance)
            run_maintenance=true
            shift
            ;;
        --snapshots)
            run_snapshots=true
            shift
            ;;
        --prefs)
            run_prefs=true
            shift
            ;;
        --login)
            run_login=true
            shift
            ;;
        --all)
            run_all=true
            shift
            ;;
        --dry-run)
            dry_run=true
            shift
            ;;
        --help | -h)
            show_optimize_help
            exit 0
            ;;
        *)
            log_warning "Unknown option: $1"
            shift
            ;;
    esac
done

# If no specific options given, show interactive menu or run all
if [[ "$run_dns" == "false" && "$run_caches" == "false" && "$run_network" == "false" && \
      "$run_maintenance" == "false" && "$run_snapshots" == "false" && \
      "$run_prefs" == "false" && "$run_login" == "false" && "$run_all" == "false" ]]; then
    run_all=true
fi

# ============================================================================
# Main Execution
# ============================================================================

echo ""
echo -e "${PURPLE_BOLD}System Optimization${NC}"
echo ""

if [[ "$dry_run" == "true" ]]; then
    echo -e "${YELLOW}[DRY RUN]${NC} No changes will be made"
    echo ""
fi

# Run selected optimizations
if [[ "$run_all" == "true" || "$run_dns" == "true" ]]; then
    if [[ "$dry_run" == "false" ]]; then
        echo -e "${BLUE}DNS Cache${NC}"
        if flush_dns_cache; then
            echo -e "  ${GREEN}${ICON_SUCCESS}${NC} DNS cache flushed"
        else
            echo -e "  ${YELLOW}!${NC} Failed to flush DNS cache"
        fi
        echo ""
    else
        echo -e "${BLUE}DNS Cache${NC} - would flush"
    fi
fi

if [[ "$run_all" == "true" || "$run_caches" == "true" ]]; then
    if [[ "$dry_run" == "false" ]]; then
        echo -e "${BLUE}System Caches${NC}"
        opt_cache_refresh
        echo ""
    else
        echo -e "${BLUE}System Caches${NC} - would refresh QuickLook, icons, Safari cache"
    fi
fi

if [[ "$run_all" == "true" || "$run_network" == "true" ]]; then
    if [[ "$dry_run" == "false" ]]; then
        echo -e "${BLUE}Network Optimization${NC}"
        opt_network_optimization
        echo ""
    else
        echo -e "${BLUE}Network Optimization${NC} - would refresh DNS, ARP, mDNS"
    fi
fi

if [[ "$run_all" == "true" || "$run_maintenance" == "true" ]]; then
    if [[ "$dry_run" == "false" ]]; then
        echo -e "${BLUE}System Maintenance${NC}"
        opt_maintenance_scripts
        echo ""
    else
        echo -e "${BLUE}System Maintenance${NC} - would rotate logs"
    fi
fi

if [[ "$run_all" == "true" || "$run_snapshots" == "true" ]]; then
    if [[ "$dry_run" == "false" ]]; then
        echo -e "${BLUE}Time Machine Snapshots${NC}"
        opt_local_snapshots
        echo ""
    else
        echo -e "${BLUE}Time Machine Snapshots${NC} - would thin local snapshots"
    fi
fi

if [[ "$run_all" == "true" || "$run_prefs" == "true" || "$run_login" == "true" ]]; then
    if [[ "$dry_run" == "false" ]]; then
        echo -e "${BLUE}System Configuration${NC}"
        opt_fix_broken_configs
        echo ""
    else
        echo -e "${BLUE}System Configuration${NC} - would fix broken prefs and login items"
    fi
fi

echo -e "${GREEN}${ICON_SUCCESS}${NC} Optimization complete"
echo ""
