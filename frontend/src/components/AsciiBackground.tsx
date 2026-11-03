import { useEffect, useRef } from "react";

const RAMP = " .:-=+*#%@";
const CELL_W = 10;
const CELL_H = 16;
const GLOW_RADIUS = 260;

export default function AsciiBackground() {
  const canvasRef = useRef<HTMLCanvasElement>(null);

  useEffect(() => {
    const canvas = canvasRef.current;
    const ctx = canvas?.getContext("2d");
    if (!canvas || !ctx) return;

    const color =
      getComputedStyle(document.documentElement)
        .getPropertyValue("--color-accent")
        .trim() || "#a78bfa";
    const reduceMotion = window.matchMedia(
      "(prefers-reduced-motion: reduce)",
    ).matches;

    let cols = 0;
    let rows = 0;
    let raf = 0;

    // pointer position in viewport space (the canvas is fixed), eased so the
    // light trails the cursor; starts off-screen so nothing glows until it moves
    let targetX = -9999;
    let targetY = -9999;
    let lightX = targetX;
    let lightY = targetY;
    const onPointerMove = (e: PointerEvent) => {
      if (lightX < -999) {
        lightX = e.clientX;
        lightY = e.clientY;
      }
      targetX = e.clientX;
      targetY = e.clientY;
    };

    const resize = () => {
      const dpr = window.devicePixelRatio || 1;
      const { clientWidth: w, clientHeight: h } = canvas;
      canvas.width = w * dpr;
      canvas.height = h * dpr;
      ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
      ctx.font = `${CELL_H - 2}px ui-monospace, monospace`;
      ctx.textBaseline = "top";
      cols = Math.ceil(w / CELL_W);
      rows = Math.ceil(h / CELL_H);
    };

    const draw = (ms: number) => {
      const t = ms / 1000;
      ctx.clearRect(0, 0, canvas.width, canvas.height);
      ctx.fillStyle = color;
      lightX += (targetX - lightX) * 0.12;
      lightY += (targetY - lightY) * 0.12;

      for (let y = 0; y < rows; y++) {
        const ny = y / rows;
        for (let x = 0; x < cols; x++) {
          const nx = x / cols;
          const v =
            Math.sin(nx * 6 + t * 0.6) +
            Math.sin(ny * 8 - t * 0.5) +
            Math.sin((nx + ny) * 5 + t * 0.4) +
            Math.sin(Math.hypot(nx - 0.5, ny - 0.5) * 12 - t * 0.8);
          // normalise to 0..1, then fade toward the top so content stays readable
          const n = (v + 4) / 8;
          const base = n * (0.35 + 0.65 * ny);
          const dist = Math.hypot(x * CELL_W - lightX, y * CELL_H - lightY);
          const glow = Math.max(0, 1 - dist / GLOW_RADIUS) ** 2;
          const level = Math.min(1, base + glow * 0.8);
          const i = Math.floor(level * RAMP.length);
          if (i <= 0) continue;
          ctx.globalAlpha = Math.min(1, 0.08 + base * 0.35 + glow * 0.7);
          ctx.fillText(RAMP[Math.min(i, RAMP.length - 1)], x * CELL_W, y * CELL_H);
        }
      }
    };

    const loop = (ms: number) => {
      draw(ms);
      raf = requestAnimationFrame(loop);
    };

    resize();
    window.addEventListener("resize", resize);
    if (!reduceMotion) window.addEventListener("pointermove", onPointerMove);
    if (reduceMotion) draw(0);
    else raf = requestAnimationFrame(loop);

    return () => {
      cancelAnimationFrame(raf);
      window.removeEventListener("resize", resize);
      window.removeEventListener("pointermove", onPointerMove);
    };
  }, []);

  return (
    <canvas
      ref={canvasRef}
      aria-hidden
      className="pointer-events-none fixed inset-0 h-full w-full"
    />
  );
}
