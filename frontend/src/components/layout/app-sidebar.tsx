import { useRef, useState } from "react"
import { Link, useLocation } from "react-router"
import { ChevronDown, ChevronRight } from "lucide-react"
import { cn } from "@/lib/utils"
import { useIsDemoMode } from "@/lib/services/demo-service"

export interface SidebarNavItem {
  label: string
  href?: string
  icon: React.ElementType
  children?: { label: string; href: string }[]
}

export interface SidebarDrawerProps {
  mobileOpen: boolean
  onMobileClose: () => void
}

interface AppSidebarProps extends SidebarDrawerProps {
  items: SidebarNavItem[]
}

// Below lg the sidebar is an off-canvas drawer opened from the header. From lg
// up it is an icon rail that widens over the content while the pointer (or
// keyboard focus) is inside it, so the page never reflows on hover.
export function AppSidebar({
  items,
  mobileOpen,
  onMobileClose,
}: AppSidebarProps) {
  const { pathname } = useLocation()
  const isDemo = useIsDemoMode()
  const [expandedItems, setExpandedItems] = useState<string[]>([])
  const [railExpanded, setRailExpanded] = useState(false)
  // Touch input fires no pointerleave once the finger lifts, so a rail opened
  // by tapping has to be closed explicitly when a link is chosen.
  const lastPointerType = useRef<string>("mouse")

  const isActive = (href: string) =>
    pathname === href || pathname.startsWith(href + "/")

  const isParentActive = (item: SidebarNavItem) => {
    if (item.href) return isActive(item.href)
    return item.children?.some((child) => isActive(child.href)) || false
  }

  const toggleExpand = (label: string) => {
    // A tap on a collapsed rail should reveal the submenu, not just flip
    // state that stays hidden until the next hover.
    if (!railExpanded) {
      setRailExpanded(true)
      setExpandedItems((prev) =>
        prev.includes(label) ? prev : [...prev, label]
      )
      return
    }
    setExpandedItems((prev) =>
      prev.includes(label)
        ? prev.filter((item) => item !== label)
        : [...prev, label]
    )
  }

  const handleNavigate = () => {
    onMobileClose()
    if (lastPointerType.current !== "mouse") setRailExpanded(false)
  }

  // Labels stay in the DOM while the rail is collapsed so screen readers and
  // the drawer (which is always wide) still get them.
  const hiddenOnRail = !railExpanded && "lg:pointer-events-none lg:opacity-0"

  // The header spans the full width, so the sidebar and its backdrop start
  // below it (and below the demo strip when that is shown).
  const belowHeader = isDemo
    ? "top-24 h-[calc(100dvh-6rem)]"
    : "top-16 h-[calc(100dvh-4rem)]"

  return (
    <>
      <div
        aria-hidden="true"
        onClick={onMobileClose}
        className={cn(
          "fixed inset-x-0 z-[35] bg-black/40 transition-opacity lg:hidden",
          belowHeader,
          mobileOpen ? "opacity-100" : "pointer-events-none opacity-0"
        )}
      />
      <aside
        aria-label="Ana menü"
        onPointerDown={(e) => {
          lastPointerType.current = e.pointerType
        }}
        onPointerEnter={(e) => {
          if (e.pointerType === "mouse") setRailExpanded(true)
        }}
        onPointerLeave={(e) => {
          if (e.pointerType === "mouse") setRailExpanded(false)
        }}
        onFocus={(e) => {
          if (e.target.matches(":focus-visible")) setRailExpanded(true)
        }}
        onBlur={(e) => {
          if (!e.currentTarget.contains(e.relatedTarget as Node | null)) {
            setRailExpanded(false)
          }
        }}
        onKeyDown={(e) => {
          if (e.key === "Escape") {
            onMobileClose()
            setRailExpanded(false)
          }
        }}
        className={cn(
          "fixed left-0 z-40 flex w-72 flex-col overflow-hidden border-r border-border bg-background transition-[width,translate,box-shadow] duration-200 ease-out",
          belowHeader,
          mobileOpen ? "translate-x-0 shadow-xl" : "-translate-x-full",
          "lg:translate-x-0",
          railExpanded ? "lg:w-64 lg:shadow-xl" : "lg:w-16 lg:shadow-none"
        )}
      >
        {/* Navigation */}
        <nav className="min-h-0 flex-1 overflow-x-hidden overflow-y-auto px-3 py-5">
          <ul className="space-y-1">
            {items.map((item) => {
              const Icon = item.icon
              const hasChildren = !!item.children && item.children.length > 0
              const isExpanded = expandedItems.includes(item.label)
              const parentActive = isParentActive(item)
              const rowClass = cn(
                "flex w-full items-center gap-3 rounded-lg px-2.5 py-2.5 text-sm font-medium transition-colors",
                parentActive
                  ? "bg-card font-semibold text-foreground shadow-xs [&>svg:first-child]:text-primary"
                  : "text-muted-foreground hover:bg-muted hover:text-foreground"
              )

              return (
                <li key={item.label}>
                  {hasChildren ? (
                    <>
                      <button
                        type="button"
                        onClick={() => toggleExpand(item.label)}
                        aria-expanded={isExpanded}
                        className={rowClass}
                      >
                        <Icon className="h-5 w-5 shrink-0" />
                        <span
                          className={cn(
                            "flex-1 truncate text-left whitespace-nowrap transition-opacity",
                            hiddenOnRail
                          )}
                        >
                          {item.label}
                        </span>
                        {isExpanded ? (
                          <ChevronDown
                            className={cn("h-4 w-4 shrink-0", hiddenOnRail)}
                          />
                        ) : (
                          <ChevronRight
                            className={cn("h-4 w-4 shrink-0", hiddenOnRail)}
                          />
                        )}
                      </button>
                      {isExpanded && item.children && (
                        <ul
                          className={cn(
                            "mt-1 ml-5 space-y-1 border-l-2 border-border pl-3",
                            !railExpanded && "lg:hidden"
                          )}
                        >
                          {item.children.map((child) => (
                            <li key={child.href}>
                              <Link
                                to={child.href}
                                onClick={handleNavigate}
                                className={cn(
                                  "block truncate rounded-lg px-3 py-2 text-sm whitespace-nowrap transition-colors",
                                  isActive(child.href)
                                    ? "bg-card font-semibold text-foreground shadow-xs"
                                    : "text-muted-foreground hover:bg-muted hover:text-foreground"
                                )}
                              >
                                {child.label}
                              </Link>
                            </li>
                          ))}
                        </ul>
                      )}
                    </>
                  ) : (
                    <Link
                      to={item.href!}
                      onClick={handleNavigate}
                      className={rowClass}
                    >
                      <Icon className="h-5 w-5 shrink-0" />
                      <span
                        className={cn(
                          "truncate whitespace-nowrap transition-opacity",
                          hiddenOnRail
                        )}
                      >
                        {item.label}
                      </span>
                    </Link>
                  )}
                </li>
              )
            })}
          </ul>
        </nav>
      </aside>
    </>
  )
}
