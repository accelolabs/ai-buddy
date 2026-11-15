export const CHAT_CIRCUIT_ANIMATION_STYLES = `
  @keyframes neon-pulse {
    0%, 100% { opacity: 0.35; filter: drop-shadow(0 0 2px currentColor); }
    50% { opacity: 1; filter: drop-shadow(0 0 6px currentColor); }
  }
  @keyframes dash-flow {
    0% { stroke-dashoffset: 1; }
    100% { stroke-dashoffset: 0; }
  }
  .animate-neon {
    animation: neon-pulse 3s infinite ease-in-out;
  }
  .animate-flow {
    stroke-dasharray: 0.3 0.7;
    animation: dash-flow 4.5s infinite linear;
  }
`;
