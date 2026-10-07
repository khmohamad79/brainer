(function (global) {
  let holds = 0;

  function render() {
    const el = document.getElementById("memobar-text");
    if (!el) return;
    el.textContent = holds > 0 ? "Memorizing ..." : "";
  }

  global.memoryHold = function () {
    holds++;
    render();
    let released = false;
    return function () {
      if (released) return;
      released = true;
      holds = Math.max(0, holds - 1);
      render();
    };
  };
})(window);
