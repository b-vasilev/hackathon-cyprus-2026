import { useEffect } from 'react';
import { Link } from 'react-router-dom';
import { setFlag, withAchievements } from '../foundations/achievements';

export function NotFound() {
  useEffect(() => {
    withAchievements(() => setFlag('off-route'));
  }, []);

  return (
    <section className="not-found" aria-labelledby="not-found-title">
      <svg viewBox="0 0 320 120" className="not-found-route" aria-hidden="true">
        <path d="M20 96 H120 V40 H200" className="route-solid" />
        <path d="M200 40 C240 40 250 90 290 70" className="route-lost" />
        <circle cx="20" cy="96" r="7" className="route-stop" />
        <circle cx="120" cy="40" r="7" className="route-stop" />
        <circle cx="200" cy="40" r="7" className="route-stop current" />
        <text x="292" y="64" className="route-question">
          ?
        </text>
      </svg>
      <p className="eyebrow">404 · Page not found</p>
      <h1 id="not-found-title">You&apos;re off the route.</h1>
      <p className="page-description">
        This page is not on the map. Even good explorers take a wrong turn sometimes. The good news:
        you just found something most people never see.
      </p>
      <div className="not-found-actions">
        <Link className="button primary" to="/foundations">
          Find your way back
        </Link>
        <Link className="button secondary" to="/">
          Product spaces
        </Link>
      </div>
    </section>
  );
}
