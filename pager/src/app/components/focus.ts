/** Contain Tab inside a container (dialog/drawer focus trap, critique P1
 *  follow-up: Tab was escaping behind aria-modal surfaces). Call from a
 *  Tab keydown handler; returns true when it handled the event. */
export function trapTab(container: HTMLElement, event: KeyboardEvent): boolean {
  const focusables = container.querySelectorAll<HTMLElement>(
    'a[href], button:not([disabled]), input:not([disabled]), select, textarea',
  );
  if (focusables.length === 0) return false;
  const first = focusables[0];
  const last = focusables[focusables.length - 1];
  const active = document.activeElement;
  const inside = active instanceof Node && container.contains(active);
  if (event.shiftKey && (!inside || active === first)) {
    event.preventDefault();
    last.focus();
    return true;
  }
  if (!event.shiftKey && (!inside || active === last)) {
    event.preventDefault();
    first.focus();
    return true;
  }
  return false;
}
