export interface CircuitTrace {
  id: string;
  path: string;
  startY: number;
  dot: { cx: number; cy: number };
  color: string;
  delay: string;
}

export const LEFT_CIRCUIT_CONFIG: CircuitTrace[] = [
  {
    id: 'left-blue-1',
    startY: 55,
    path: 'M 0 55 H 250 C 275 55 290 70 290 95 V 140 C 290 160 305 175 325 175 H 410',
    dot: { cx: 410, cy: 175 },
    color: '#0EA5E9',
    delay: '0s',
  },
  {
    id: 'left-green-1',
    startY: 135,
    path: 'M 0 135 H 150 C 175 135 190 150 190 175 V 230 C 190 250 205 265 225 265 H 290',
    dot: { cx: 290, cy: 265 },
    color: '#10B981',
    delay: '1.4s',
  },
  {
    id: 'left-purple-1',
    startY: 240,
    path: 'M 0 240 H 260 C 285 240 300 225 300 200 V 170',
    dot: { cx: 300, cy: 170 },
    color: '#8B5CF6',
    delay: '2.6s',
  },
  {
    id: 'left-blue-2',
    startY: 350,
    path: 'M 0 350 H 210 C 235 350 250 365 250 390 V 460 C 250 480 265 495 285 495 H 380',
    dot: { cx: 380, cy: 495 },
    color: '#0EA5E9',
    delay: '0.8s',
  },
  {
    id: 'left-green-2',
    startY: 470,
    path: 'M 0 470 H 130 C 155 470 170 485 170 510 V 580 C 170 600 185 615 205 615 H 240',
    dot: { cx: 240, cy: 615 },
    color: '#10B981',
    delay: '2.1s',
  },
  {
    id: 'left-red-1',
    startY: 575,
    path: 'M 0 575 H 240 C 265 575 280 590 280 615 V 680 C 280 700 295 715 315 715 H 395',
    dot: { cx: 395, cy: 715 },
    color: '#F43F5E',
    delay: '3.2s',
  },
  {
    id: 'left-blue-3',
    startY: 690,
    path: 'M 0 690 H 170 C 195 690 210 705 210 730 V 790',
    dot: { cx: 210, cy: 790 },
    color: '#0EA5E9',
    delay: '1.7s',
  },
  {
    id: 'left-purple-2',
    startY: 780,
    path: 'M 0 780 H 110 C 130 780 145 765 145 745 V 720',
    dot: { cx: 145, cy: 720 },
    color: '#8B5CF6',
    delay: '2.9s',
  },
];

export const RIGHT_CIRCUIT_CONFIG: CircuitTrace[] = [
  {
    id: 'right-purple-1',
    startY: 85,
    path: 'M 340 85 H 180 C 155 85 140 100 140 125 V 195',
    dot: { cx: 140, cy: 195 },
    color: '#8B5CF6',
    delay: '0.9s',
  },
  {
    id: 'right-red-1',
    startY: 270,
    path: 'M 340 270 H 190 C 165 270 150 285 150 310 V 380 C 150 400 135 415 115 415 H 60',
    dot: { cx: 60, cy: 415 },
    color: '#F43F5E',
    delay: '2.2s',
  },
  {
    id: 'right-green-1',
    startY: 540,
    path: 'M 340 540 H 170 C 145 540 130 555 130 580 V 670',
    dot: { cx: 130, cy: 670 },
    color: '#10B981',
    delay: '0.4s',
  },
  {
    id: 'right-blue-1',
    startY: 680,
    path: 'M 340 680 H 220 C 195 680 180 665 180 640 V 605',
    dot: { cx: 180, cy: 605 },
    color: '#0EA5E9',
    delay: '1.8s',
  },
];
