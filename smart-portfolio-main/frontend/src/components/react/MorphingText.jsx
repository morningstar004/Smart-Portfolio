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
    <div className="relative w-full h-37.5 flex items-center justify-start overflow-visible max-lg:h-28 max-md:h-20 max-sm:h-12">
      {/* Refined SVG Filter Definitions */}
      <svg className="absolute h-0 w-0" aria-hidden="true" focusable="false">
        <defs>
          {/* Desktop Filter - stdDeviation 3.5 */}
          <filter id="goo">
            <feGaussianBlur in="SourceGraphic" stdDeviation="3.5" result="blur" />
            <feColorMatrix 
              in="blur" 
              mode="matrix" 
              values="1 0 0 0 0  0 1 0 0 0  0 0 1 0 0  0 0 0 255 -140" 
              result="goo" 
            />
          </filter>

          {/* Tablet Filter - stdDeviation 2.3 */}
          <filter id="goo-tablet">
            <feGaussianBlur in="SourceGraphic" stdDeviation="2.2" result="blur" />
            <feColorMatrix 
              in="blur" 
              mode="matrix" 
              values="1 0 0 0 0  0 1 0 0 0  0 0 1 0 0  0 0 0 255 -140" 
              result="goo" 
            />
          </filter>

          {/* Mobile Filter - stdDeviation 1.8 */}
          <filter id="goo-mobile">
            <feGaussianBlur in="SourceGraphic" stdDeviation="1.2" result="blur" />
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
        /* 
           Appended CSS variables to handle responsive filter switching 
           while strictly preserving the style property for the desktop truth.
        */
        className="relative w-full h-full [--goo-filter:url(#goo-mobile)] md:[--goo-filter:url(#goo-tablet)] lg:[--goo-filter:url(#goo)]"
        style={{ filter: "var(--goo-filter)" }}
      >
        {words.map((word, i) => (
          <span
            key={word}
            className={`
              absolute left-0 top-1/2 -translate-y-1/2 text-black
              text-[105px] font-exterbold uppercase tracking-tighter leading-none
              transition-all duration-1800 ease-linear whitespace-nowrap 
              max-lg:text-[75px] max-md:text-[50px] max-sm:text-[28px]
              ${i === index ? 'opacity-100 scale-100' : 'opacity-0 scale-100'}
            `}
            style={{ 
                fontFamily: "'M PLUS Rounded 1c', sans-serif",
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