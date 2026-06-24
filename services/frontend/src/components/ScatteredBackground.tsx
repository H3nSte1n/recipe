import '../styles/ScatteredBackground.css';

const tiles = [
  { left: '10.68%', top: '19.07%', size: 180 },
  { left: '-5.31%', top: '35.65%', size: 223 },
  { left: '8.75%',  top: '70.83%', size: 180 },
  { left: '22.34%', top: '46.67%', size: 144 },
  { left: '27.19%', top: '0.56%',  size: 180 },
  { left: '34.53%', top: '74.91%', size: 180 },
  { left: '40.16%', top: '26.11%', size: 90  },
  { left: '55.99%', top: '22.87%', size: 97  },
  { left: '69.43%', top: '35.56%', size: 126 },
  { left: '76.93%', top: '9.44%',  size: 180 },
  { left: '65.26%', top: '69.72%', size: 180 },
  { left: '86.46%', top: '64.72%', size: 279 },
];

export default function ScatteredBackground() {
  return (
    <div className="scattered-bg" aria-hidden="true">
      {tiles.map((tile, i) => (
        <div
          key={i}
          className="scattered-bg__tile"
          style={{ left: tile.left, top: tile.top, width: tile.size, height: tile.size }}
        />
      ))}
      <div className="scattered-bg__blur" />
    </div>
  );
}
