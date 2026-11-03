import { useEffect, useRef } from "react";

const RAMP = " .:-=+*#%@";
const CELL_W = 5;
const CELL_H = 9;
const TILT = -0.42;
const RING_SQUASH = 0.28;
const LIGHT = { x: -0.55, y: -0.45, z: 0.7 };

export default function AsciiSaturn({ className }: { className?: string }) {
  const canvasRef = useRef<HTMLCanvasElement>(null);

  useEffect(() => {
    const canvas = canvasRef.current;
    const ctx = canvas?.getContext("2d");
    if (!canvas || !ctx) return;

    const css = getComputedStyle(document.documentElement);
    const color = css.getPropertyValue("--color-accent-light").trim() || "#c4b5fd";
    // painted behind Saturn's own cells so the background can't show through
    const panel = css.getPropertyValue("--color-panel").trim() || "#000";
    const reduceMotion = window.matchMedia(
      "(prefers-reduced-motion: reduce)",
    ).matches;

    let w = 0;
    let h = 0;
    let raf = 0;
    const cos = Math.cos(TILT);
    const sin = Math.sin(TILT);

    const resize = () => {
      const dpr = window.devicePixelRatio || 1;
      w = canvas.clientWidth;
      h = canvas.clientHeight;
      canvas.width = w * dpr;
      canvas.height = h * dpr;
      ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
      ctx.font = `${CELL_H - 1}px ui-monospace, monospace`;
      ctx.textBaseline = "top";
    };

    const draw = (ms: number) => {
      const t = ms / 1000;
      ctx.clearRect(0, 0, w, h);

      const R = Math.min(w / 4.6, h / 2.4);
      const cols = Math.ceil(w / CELL_W);
      const rows = Math.ceil(h / CELL_H);

      for (let y = 0; y < rows; y++) {
        for (let x = 0; x < cols; x++) {
          // pixel offset from centre, rotated into the ring's frame
          const dx = x * CELL_W + CELL_W / 2 - w / 2;
          const dy = y * CELL_H + CELL_H / 2 - h / 2;
          const u = (dx * cos + dy * sin) / R;
          const v = (-dx * sin + dy * cos) / R;

          const d2 = u * u + v * v;
          const onPlanet = d2 < 1;
          const rr = Math.sqrt(u * u + (v / RING_SQUASH) ** 2);
          const inBand = rr > 1.3 && rr < 2.3;
          const onRing = inBand && !(rr > 1.78 && rr < 1.9);
          // the far half of the ring is hidden behind the planet
          const ringVisible = onRing && (v > 0 || !onPlanet);
          // the gap is empty but still masks the background
          const masked = onPlanet || (inBand && (v > 0 || !onPlanet));
          if (!masked) continue;

          let level = 0;
          if (ringVisible) {
            const grain = 0.5 + 0.5 * Math.sin(rr * 38 - t * 0.8);
            level = (rr < 1.78 ? 0.55 : 0.4) + 0.3 * grain;
            // soft shadow cast by the planet onto the far ring
            if (v < 0 && Math.abs(u) < 1.0) level *= 0.6;
          } else if (onPlanet) {
            const nz = Math.sqrt(1 - d2);
            const lon = Math.atan2(u, nz) + t * 0.35;
            const bands =
              0.5 + 0.5 * Math.sin(v * 13 + Math.sin(lon * 3 + v * 4) * 0.35);
            const diffuse = Math.max(0, u * LIGHT.x + v * LIGHT.y + nz * LIGHT.z);
            level = 0.08 + diffuse * (0.45 + 0.45 * bands);
          }

          ctx.globalAlpha = 1;
          ctx.fillStyle = panel;
          ctx.fillRect(x * CELL_W, y * CELL_H, CELL_W + 0.5, CELL_H + 0.5);

          const i = Math.floor(Math.min(level, 0.999) * RAMP.length);
          if (i <= 0) continue;
          ctx.fillStyle = color;
          ctx.globalAlpha = 0.35 + 0.65 * (i / RAMP.length);
          ctx.fillText(RAMP[i], x * CELL_W, y * CELL_H);
        }
      }
    };

    const loop = (ms: number) => {
      draw(ms);
      raf = requestAnimationFrame(loop);
    };

    resize();
    window.addEventListener("resize", resize);
    if (reduceMotion) draw(0);
    else raf = requestAnimationFrame(loop);

    return () => {
      cancelAnimationFrame(raf);
      window.removeEventListener("resize", resize);
    };
  }, []);

  return (
    <canvas
      ref={canvasRef}
      aria-hidden
      className={className ?? "pointer-events-none h-full w-full"}
    />
  );
}
