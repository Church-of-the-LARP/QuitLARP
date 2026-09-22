import { useEffect, useRef } from "react";

const DUNE_COLORS = {
  back: "oklch(86.5% 0.012 325.68)", // mauve-300
  mid1: "oklch(71.1% 0.019 323.02)", // mauve-400
  mid2: "oklch(54.2% 0.034 322.5)", // mauve-500
  front: "oklch(36.4% 0.029 323.89)", // mauve-700
};

const SUN_CORE = "oklch(96% 0.025 100)";

const EASE = 0.055;
const IDLE_EASE = 0.02;
const SKY_MIN_Y = 40;
const SKY_MAX_Y_RATIO = 0.82;

export default function DesertScene() {
  const containerRef = useRef<HTMLDivElement>(null);
  const sunWrapRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const container = containerRef.current;
    const sunWrap = sunWrapRef.current;
    if (!container || !sunWrap) return;

    const reduced = window.matchMedia(
      "(prefers-reduced-motion: reduce)"
    ).matches;
    const canHover = window.matchMedia("(hover: hover)").matches;

    // Target/current position are tracked in viewport space (matching
    // clientX/clientY); only converted to container-local coordinates at
    // the moment we write the transform, using a fresh rect each frame so
    // scrolling/resizing the page keeps the sun correctly placed.
    let targetX = window.innerWidth * 0.5;
    let targetY = window.innerHeight * 0.3;
    let sunX = targetX;
    let sunY = targetY;
    let rafId: number;

    const clampY = (y: number, top: number, height: number) => {
      const minY = top + SKY_MIN_Y;
      const maxY = top + height * SKY_MAX_Y_RATIO;
      return Math.max(minY, Math.min(maxY, y));
    };

    const writePosition = () => {
      const rect = container.getBoundingClientRect();
      sunWrap.style.transform = `translate(${sunX - rect.left}px, ${
        sunY - rect.top
      }px)`;
    };

    if (reduced) {
      const rect = container.getBoundingClientRect();
      targetY = clampY(targetY, rect.top, rect.height);
      sunX = targetX;
      sunY = targetY;
      writePosition();
      return;
    }

    if (canHover) {
      const onPointerMove = (e: PointerEvent) => {
        const rect = container.getBoundingClientRect();
        targetX = e.clientX;
        targetY = clampY(e.clientY, rect.top, rect.height);
      };
      window.addEventListener("pointermove", onPointerMove);

      const animate = () => {
        sunX += (targetX - sunX) * EASE;
        sunY += (targetY - sunY) * EASE;
        writePosition();
        rafId = requestAnimationFrame(animate);
      };
      rafId = requestAnimationFrame(animate);

      return () => {
        window.removeEventListener("pointermove", onPointerMove);
        cancelAnimationFrame(rafId);
      };
    }

    const t0 = performance.now();
    const idle = (t: number) => {
      const dt = (t - t0) / 1000;
      const rect = container.getBoundingClientRect();
      targetX = rect.left + rect.width * 0.5 + Math.sin(dt * 0.15) * rect.width * 0.28;
      targetY = clampY(
        rect.top + rect.height * 0.3 + Math.sin(dt * 0.1) * rect.height * 0.08,
        rect.top,
        rect.height
      );
      sunX += (targetX - sunX) * IDLE_EASE;
      sunY += (targetY - sunY) * IDLE_EASE;
      writePosition();
      rafId = requestAnimationFrame(idle);
    };
    rafId = requestAnimationFrame(idle);

    return () => cancelAnimationFrame(rafId);
  }, []);

  return (
    <div
      className="desert-scene"
      aria-hidden="true"
      ref={containerRef}
      style={
        {
          "--dune-back": DUNE_COLORS.back,
          "--dune-mid1": DUNE_COLORS.mid1,
          "--dune-mid2": DUNE_COLORS.mid2,
          "--dune-front": DUNE_COLORS.front,
          "--sun-core": SUN_CORE,
        } as React.CSSProperties
      }
    >
      <style>{`
        .desert-scene {
          position: absolute;
          inset: 0;
          z-index: 0;
          overflow: hidden;
          pointer-events: none;
        }

        .desert-scene .sun-wrap {
          position: absolute;
          top: 0;
          left: 0;
          width: 0;
          height: 0;
          z-index: 1;
          will-change: transform;
        }

        .desert-scene .sun-halo {
          position: absolute;
          width: 1100px;
          height: 1100px;
          margin-left: -550px;
          margin-top: -550px;
          border-radius: 50%;
          background: radial-gradient(circle at 50% 50%,
            color-mix(in oklch, var(--color-accent-light) 70%, transparent) 0%,
            color-mix(in oklch, var(--color-accent) 38%, transparent) 34%,
            transparent 68%);
          mix-blend-mode: screen;
          animation: desert-scene-breathe 7s ease-in-out infinite;
        }

        .desert-scene .sun-core {
          position: absolute;
          width: 220px;
          height: 220px;
          margin-left: -110px;
          margin-top: -110px;
          border-radius: 50%;
          background: radial-gradient(circle at 50% 50%,
            var(--sun-core) 0%,
            var(--sun-core) 22%,
            var(--color-accent-light) 48%,
            var(--color-accent) 70%,
            transparent 80%);
          box-shadow:
            0 0 90px 30px color-mix(in oklch, var(--color-accent-light) 75%, transparent),
            0 0 220px 90px color-mix(in oklch, var(--color-accent) 45%, transparent);
          mix-blend-mode: screen;
        }

        @keyframes desert-scene-breathe {
          0%, 100% { transform: scale(1); opacity: 0.95; }
          50% { transform: scale(1.1); opacity: 1; }
        }

        @media (prefers-reduced-motion: reduce) {
          .desert-scene .sun-halo { animation: none; }
        }

        .desert-scene .dunes {
          position: absolute;
          left: 0;
          right: 0;
          bottom: -1px;
          width: 100%;
          display: block;
        }

        /* Only the furthest layer fades into the sky; everything closer
           is flat, fully opaque color with a crisp silhouette edge, like
           layered cutouts, so each mound reads as a distinct shape rather
           than everything blending into one soft gradient. The fade is a
           CSS mask (not an SVG <linearGradient> fill) because a gradient
           fill combined with preserveAspectRatio="none" under heavy
           non-uniform scaling triggers a real browser rasterization bug
           (stray dark triangles cut into the shape); a solid fill +
           mask-image is composited after the SVG is rasterized, which
           sidesteps it. */
        .desert-scene .dune-back {
          height: 62vh;
          z-index: 2;
          mask-image: linear-gradient(to bottom, transparent 0%, black 55%);
          -webkit-mask-image: linear-gradient(to bottom, transparent 0%, black 55%);
        }
        .desert-scene .dune-mid-1 { height: 50vh; z-index: 3; }
        .desert-scene .dune-mid-2 { height: 38vh; z-index: 4; }
        .desert-scene .dune-front { height: 27vh; z-index: 5; }
      `}</style>

      <div className="sun-wrap" ref={sunWrapRef}>
        <div className="sun-halo" />
        <div className="sun-core" />
      </div>

      <svg
        className="dunes dune-back"
        viewBox="0 0 1440 558"
        preserveAspectRatio="none"
      >
        <path
          d="M0,260 C379.5,223.7 632.5,150 1150,150 C1280.5,150 1344.3,196.9 1440,220 L1440,558 L0,558 Z"
          fill="var(--dune-back)"
        />
      </svg>

      <svg
        className="dunes dune-mid-1"
        viewBox="0 0 1440 450"
        preserveAspectRatio="none"
      >
        <path
          d="M0,300 C165,263.7 290,190 500,190 C647,190 703,290 850,290 C1018,290 1082,230 1250,230 C1329.8,230 1377.3,263.5 1440,280 L1440,450 L0,450 Z"
          fill="var(--dune-mid1)"
        />
      </svg>

      <svg
        className="dunes dune-mid-2"
        viewBox="0 0 1440 342"
        preserveAspectRatio="none"
      >
        <path
          d="M0,290 C92.4,266.9 162.4,220 280,220 C393.4,220 436.6,300 550,300 C747.4,300 822.6,160 1020,160 C1196.4,160 1301.4,220.3 1440,250 L1440,342 L0,342 Z"
          fill="var(--dune-mid2)"
        />
      </svg>

      <svg
        className="dunes dune-front"
        viewBox="0 0 1440 243"
        preserveAspectRatio="none"
      >
        <path
          d="M0,205 C115.5,186.85 203,150 350,150 C476,150 524,195 650,195 C776,195 824,120 950,120 C1155.8,120 1278.3,156.85 1440,175 L1440,243 L0,243 Z"
          fill="var(--dune-front)"
        />
      </svg>
    </div>
  );
}
