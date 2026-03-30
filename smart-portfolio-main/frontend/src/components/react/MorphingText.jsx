import React, { useState, useEffect } from 'react';

const MorphingText = () => {
  const words = ["FRONTEND DEVELOPER", "FULL-STACK DEVELOPER"];
  const [index, setIndex] = useState(0);

  useEffect(() => {
    const interval = setInterval(() => {
      setIndex((prev) => (prev + 1) % words.length);
    }, 4000); 
    return () => clearInterval(interval);
  }, []);

  return (
    <div className="relative w-full h-37.5 flex items-center justify-start overflow-visible">
      {/* Refined SVG Filter */}
      <svg className="absolute h-0 w-0" aria-hidden="true" focusable="false">
        <defs>
          <filter id="goo">
            {/* Lowered stdDeviation so the text remains crisp when static */}
            <feGaussianBlur in="SourceGraphic" stdDeviation="3.5" result="blur" />
            {/* Standardized color matrix for a cleaner alpha threshold */}
            <feColorMatrix 
              in="blur" 
              mode="matrix" 
              values="1 0 0 0 0  0 1 0 0 0  0 0 1 0 0  0 0 0 255 -140" 
              result="goo" 
            />
          </filter>
        </defs>
      </svg>

      <div 
        className="relative w-full h-full"
        style={{ filter: "url(#goo)" }}
      >
        {words.map((word, i) => (
          <span
            key={word}
            className={`
              absolute left-0 top-1/2 -translate-y-1/2 text-black
              text-[105px] font-exterbold uppercase tracking-tighter leading-none
              transition-all duration-1800 ease-linear whitespace-nowrap
              ${i === index ? 'opacity-100 scale-100' : 'opacity-0 scale-100'}
            `}
            style={{ 
                fontFamily: "'M PLUS Rounded 1c', sans-serif",
                // Removed the visibility toggle so the fade transition can actually happen
            }}
          >
            {word}
          </span>
        ))}
      </div>
    </div>
  );
};

export default MorphingText;