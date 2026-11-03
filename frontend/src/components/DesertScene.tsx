import AsciiBackground from "./AsciiBackground";
import AsciiDunes from "./AsciiDunes";
import AsciiSaturn from "./AsciiSaturn";

export default function DesertScene() {
  return (
    <div className="desert-scene" aria-hidden="true">
      <style>{`
        .desert-scene {
          position: absolute;
          inset: 0;
          z-index: 0;
          overflow: hidden;
          pointer-events: none;
        }

        .desert-scene .ascii-bg {
          position: absolute;
          inset: 0;
          z-index: 1;
        }

        .desert-scene .saturn {
          position: absolute;
          top: 6%;
          left: 50%;
          width: min(680px, 94vw);
          height: min(360px, 50vw);
          transform: translateX(-50%);
          z-index: 1;
        }
      `}</style>

      <div className="ascii-bg">
        <AsciiBackground />
      </div>

      <div className="saturn">
        <AsciiSaturn />
      </div>

      <AsciiDunes />
    </div>
  );
}
