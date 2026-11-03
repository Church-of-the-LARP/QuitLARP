import { useEffect, useRef } from "react";

const CELL_W = 5;
const CELL_H = 9;
const RAMP = ".:-=+*#%@";

// back → front. `frac` is the layer height as a fraction of the viewport.
const LAYERS = [
  {
    color: "oklch(86.5% 0.012 325.68)", // mauve-300
    frac: 0.62,
    vbH: 558,
    d: "M0,260 C379.5,223.7 632.5,150 1150,150 C1280.5,150 1344.3,196.9 1440,220 L1440,558 L0,558 Z",
  },
  {
    color: "oklch(71.1% 0.019 323.02)", // mauve-400
    frac: 0.5,
    vbH: 450,
    d: "M0,300 C165,263.7 290,190 500,190 C647,190 703,290 850,290 C1018,290 1082,230 1250,230 C1329.8,230 1377.3,263.5 1440,280 L1440,450 L0,450 Z",
  },
  {
    color: "oklch(54.2% 0.034 322.5)", // mauve-500
    frac: 0.38,
    vbH: 342,
    d: "M0,290 C92.4,266.9 162.4,220 280,220 C393.4,220 436.6,300 550,300 C747.4,300 822.6,160 1020,160 C1196.4,160 1301.4,220.3 1440,250 L1440,342 L0,342 Z",
  },
  {
    color: "oklch(36.4% 0.029 323.89)", // mauve-700
    frac: 0.27,
    vbH: 243,
    d: "M0,205 C115.5,186.85 203,150 350,150 C476,150 524,195 650,195 C776,195 824,120 950,120 C1155.8,120 1278.3,156.85 1440,175 L1440,243 L0,243 Z",
  },
].map((l) => ({ ...l, path: new Path2D(l.d) }));

export default function AsciiDunes() {
  const canvasRef = useRef<HTMLCanvasElement>(null);

  useEffect(() => {
    const canvas = canvasRef.current;
    const ctx = canvas?.getContext("2d");
    if (!canvas || !ctx) return;

    // painted behind every dune cell so the background can't show through
    const panel =
      getComputedStyle(document.documentElement)
        .getPropertyValue("--color-panel")
        .trim() || "#000";
    const reduceMotion = window.matchMedia(
      "(prefers-reduced-motion: reduce)",
    ).matches;

    let cols = 0;
    let rows = 0;
    let owner = new Int8Array(0);
    let depth = new Int16Array(0);
    let raf = 0;

    const layout = () => {
      const dpr = window.devicePixelRatio || 1;
      const w = canvas.clientWidth;
      const h = canvas.clientHeight;
      if (!w || !h) return;
      canvas.width = w * dpr;
      canvas.height = h * dpr;
      ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
      ctx.font = `${CELL_H - 1}px ui-monospace, monospace`;
      ctx.textBaseline = "top";

      cols = Math.ceil(w / CELL_W);
      rows = Math.ceil(h / CELL_H);

      // rasterise each layer at one pixel per character cell, so pixel
      // coverage == which characters belong to that layer
      const mask = document.createElement("canvas");
      mask.width = cols;
      mask.height = rows;
      const m = mask.getContext("2d", { willReadFrequently: true })!;
      owner = new Int8Array(cols * rows).fill(-1);
      LAYERS.forEach((l, i) => {
        m.setTransform(1, 0, 0, 1, 0, 0);
        m.clearRect(0, 0, cols, rows);
        const boxH = l.frac * window.innerHeight;
        m.setTransform(
          w / 1440 / CELL_W,
          0,
          0,
          boxH / l.vbH / CELL_H,
          0,
          (h - boxH) / CELL_H,
        );
        m.fill(l.path);
        const px = m.getImageData(0, 0, cols, rows).data;
        for (let k = 0; k < owner.length; k++) if (px[k * 4 + 3] > 127) owner[k] = i;
      });

      // distance below the crest, used to emphasise the ridge line
      depth = new Int16Array(cols * rows);
      for (let y = 1; y < rows; y++) {
        for (let x = 0; x < cols; x++) {
          const k = y * cols + x;
          if (owner[k] >= 0 && owner[k - cols] === owner[k]) depth[k] = depth[k - cols] + 1;
        }
      }
    };

    const draw = (ms: number) => {
      const t = ms / 1000;
      ctx.clearRect(0, 0, canvas.width, canvas.height);

      for (let y = 0; y < rows; y++) {
        for (let x = 0; x < cols; x++) {
          const k = y * cols + x;
          const o = owner[k];
          if (o < 0) continue;

          // wind ripples drifting across each layer; nearer layers move faster
          const phase = t * (0.5 + o * 0.35);
          const ripple =
            0.5 +
            0.5 *
              Math.sin(
                x * 0.22 - phase + y * 0.55 + Math.sin(x * 0.05 + o + phase * 0.3) * 3 + o * 2,
              );
          const crest = Math.max(0, 1 - depth[k] / 5);
          const level = ripple * 0.6 + crest * 0.4;

          ctx.fillStyle = panel;
          ctx.fillRect(x * CELL_W, y * CELL_H, CELL_W + 0.5, CELL_H + 0.5);
          ctx.fillStyle = LAYERS[o].color;
          ctx.fillText(
            RAMP[Math.min(RAMP.length - 1, Math.floor(level * RAMP.length))],
            x * CELL_W,
            y * CELL_H,
          );
        }
      }
    };

    const loop = (ms: number) => {
      draw(ms);
      raf = requestAnimationFrame(loop);
    };

    const start = () => {
      cancelAnimationFrame(raf);
      layout();
      if (reduceMotion) draw(0);
      else raf = requestAnimationFrame(loop);
    };

    start();
    const ro = new ResizeObserver(start);
    ro.observe(canvas);
    return () => {
      cancelAnimationFrame(raf);
      ro.disconnect();
    };
  }, []);

  return (
    <canvas
      ref={canvasRef}
      aria-hidden
      className="pointer-events-none absolute inset-0 z-2 h-full w-full"
    />
  );
}
