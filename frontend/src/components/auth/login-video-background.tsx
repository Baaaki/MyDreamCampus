import { useState } from "react"

const VIDEO_SRC = "/videos/anakampus_web.mp4"
const POSTER_SRC = "/videos/anakampus_web-poster.jpg"

// Reduced-motion visitors get the still frame only: no <video> element means
// the large file is never downloaded either.
function prefersReducedMotion() {
  return (
    window.matchMedia?.("(prefers-reduced-motion: reduce)").matches ?? false
  )
}

export function LoginVideoBackground() {
  const [reducedMotion] = useState(prefersReducedMotion)

  return (
    <div
      aria-hidden="true"
      className="pointer-events-none fixed inset-0 z-0 overflow-hidden bg-background"
    >
      {reducedMotion ? (
        <img src={POSTER_SRC} alt="" className="h-full w-full object-cover" />
      ) : (
        <video
          autoPlay
          loop
          muted
          playsInline
          preload="auto"
          poster={POSTER_SRC}
          className="h-full w-full object-cover"
        >
          <source src={VIDEO_SRC} type="video/mp4" />
        </video>
      )}
      {/* Tinted with the theme background so the glass card reads in both themes. */}
      <div className="absolute inset-0 bg-gradient-to-b from-background/70 via-background/40 to-background/80" />
    </div>
  )
}
