import { Container, Button } from "../components/ui";
import { Labyrinth } from "../components/Labyrinth";

export default function NotFound() {
  return (
    <Container className="flex min-h-[70vh] flex-col items-center justify-center gap-6 text-center">
      <Labyrinth className="h-24 w-24" strokeWidth={1.4} />
      <p className="font-mono text-sm uppercase tracking-[0.2em] text-forge-400">
        404 — lost in the labyrinth
      </p>
      <h1 className="font-display text-4xl font-semibold text-ink-50">
        This path leads nowhere
      </h1>
      <p className="max-w-md text-ink-400">
        The page you're looking for doesn't exist. Let's get you back on the
        thread.
      </p>
      <div className="flex flex-wrap items-center justify-center gap-3">
        <Button to="/" iconRight="arrow">
          Back home
        </Button>
        <Button to="/docs" variant="secondary" icon="book">
          Read the docs
        </Button>
      </div>
    </Container>
  );
}
