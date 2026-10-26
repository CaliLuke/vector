/** jsdom has no layout; give real positioners a viewport and an off-origin trigger. */
export function setPositioningGeometry(trigger) {
  Object.defineProperties(document.documentElement, {
    clientWidth: { configurable: true, value: 1200 },
    clientHeight: { configurable: true, value: 800 },
  });
  trigger.getBoundingClientRect = () => new window.DOMRect(400, 100, 200, 32);
}
