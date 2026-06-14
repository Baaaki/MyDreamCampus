import { describe, it, expect, beforeEach, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { ErrorBoundary } from "./error-boundary";

function Boom({ when }: { when: boolean }) {
  if (when) throw new Error("explosion-message-xyz");
  return <div>healthy</div>;
}

describe("ErrorBoundary", () => {
  beforeEach(() => {
    // ErrorBoundary logs to console.error in componentDidCatch — silence it.
    vi.spyOn(console, "error").mockImplementation(() => {});
  });

  it("renders children when there is no error", () => {
    render(
      <ErrorBoundary>
        <Boom when={false} />
      </ErrorBoundary>
    );
    expect(screen.getByText("healthy")).toBeInTheDocument();
  });

  it("catches a thrown error and renders the fallback", () => {
    render(
      <ErrorBoundary>
        <Boom when={true} />
      </ErrorBoundary>
    );
    expect(screen.getByText("Bir şeyler ters gitti")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Tekrar dene" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Ana sayfa" })).toBeInTheDocument();
  });

  it("shows error message in dev mode", () => {
    render(
      <ErrorBoundary>
        <Boom when={true} />
      </ErrorBoundary>
    );
    // Vitest sets DEV=true by default
    expect(screen.getByText("explosion-message-xyz")).toBeInTheDocument();
  });

  it("reset button clears the error so children can re-render", () => {
    // Rerender with a non-throwing child first so the next reset
    // re-render doesn't immediately throw and bounce back into error state.
    const { rerender } = render(
      <ErrorBoundary>
        <Boom when={true} />
      </ErrorBoundary>
    );
    expect(screen.getByText("Bir şeyler ters gitti")).toBeInTheDocument();

    rerender(
      <ErrorBoundary>
        <Boom when={false} />
      </ErrorBoundary>
    );
    // Boundary still shows fallback because state.error is still set
    expect(screen.getByText("Bir şeyler ters gitti")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Tekrar dene" }));
    expect(screen.getByText("healthy")).toBeInTheDocument();
  });
});
