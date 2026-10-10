// Small DOM helpers shared by the Ops views (FEAT-052). Only `el` survives:
// the skeleton/statusLine builders went with the retired vanilla renderers
// (TASK-pD575FN).

export function el<K extends keyof HTMLElementTagNameMap>(tag: K, className?: string, text?: string): HTMLElementTagNameMap[K] {
  const node = document.createElement(tag);
  if (className) node.className = className;
  if (text !== undefined) node.textContent = text;
  return node;
}
