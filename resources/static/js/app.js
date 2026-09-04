// Swap 4xx/5xx responses carrying an explicit HX-Retarget (server-rendered
// error toasts) — htmx only swaps 2xx by default. Loaded with defer after the
// htmx script, so both the DOM and htmx's listeners exist when this attaches.
document.body.addEventListener('htmx:beforeSwap', function (e) {
  var retarget = e.detail.xhr.getResponseHeader('HX-Retarget');
  if ((e.detail.xhr.status >= 400 || e.detail.isError) && retarget) {
    e.detail.shouldSwap = true;
    e.detail.isError = false;
  }
});

// Live-filter the collections picker's checkbox list client-side — the list is
// preloaded into the row, so searching it needs no requests.
document.addEventListener('input', function (e) {
  if (e.target.type !== 'search') return;
  var panel = e.target.closest('.edit-panel');
  if (!panel) return;
  var q = e.target.value.trim().toLowerCase();
  panel.querySelectorAll('.pick-list li').forEach(function (li) {
    li.hidden = q !== '' && !li.textContent.toLowerCase().includes(q);
  });
});