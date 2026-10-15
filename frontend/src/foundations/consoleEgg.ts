import { setFlag, withAchievements } from './achievements';

declare global {
  // Console command for curious engineers.
  var wayfinder: { explore: () => string } | undefined;
}

// A greeting for people who open DevTools, as the HTTP lesson suggests.
export function installConsoleEgg() {
  globalThis.wayfinder = {
    explore() {
      withAchievements(() => setFlag('curious'));
      return 'Achievement unlocked: Curious engineer. Now close DevTools and go finish a lesson.';
    },
  };
  // eslint-disable-next-line no-console -- the greeting is the easter egg
  console.info(
    '%cW%c Wayfinder\n\n%cHi, curious engineer. You opened DevTools: that was lesson 1 of HTTP and APIs.\nType %cwayfinder.explore()%c and press Enter.',
    'color:#fdfcfd;background:#6550b9;font-weight:700;font-size:18px;padding:2px 8px;border-radius:6px',
    'color:#6550b9;font-weight:600;font-size:16px',
    'color:inherit;font-size:13px',
    'color:#6550b9;font-family:monospace;font-size:13px;font-weight:600',
    'color:inherit;font-size:13px'
  );
}
