import React, { useState, useEffect } from 'react';
import '@/styles/Clock.css'; // Assuming you put the CSS here

const AnalogClock = () => {
  const [theme, setTheme] = useState('dark');
  const [rotations, setRotations] = useState({ hr: 0, min: 0, sec: 0 });

  // Update Clock logic
  useEffect(() => {
    const updateClock = () => {
      const day = new Date();
      const hh = day.getHours() * 30;
      const mm = day.getMinutes() * 6;
      const ss = day.getSeconds() * 6;

      setRotations({
        hr: hh + mm / 12,
        min: mm,
        sec: ss,
      });
    };

    updateClock(); // Initial call
    const interval = setInterval(updateClock, 1000);

    return () => clearInterval(interval); // Cleanup on unmount
  }, []);

  // Theme logic
  useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme);
  }, [theme]);

  const toggleTheme = () => {
    setTheme((prevTheme) => (prevTheme === 'light' ? 'dark' : 'light'));
  };

  return (
    <div className="clock-container">
      <div className="page-header">Analog Clock</div>
      
      <div className="clock">
        <div 
          className="hour" 
          style={{ transform: `rotateZ(${rotations.hr}deg)` }}
        ></div>
        <div 
          className="min" 
          style={{ transform: `rotateZ(${rotations.min}deg)` }}
        ></div>
        <div 
          className="sec" 
          style={{ transform: `rotateZ(${rotations.sec}deg)` }}
        ></div>
      </div>

      <div className="switch-cont">
        <button className="switch-btn" onClick={toggleTheme}>
          {theme === 'light' ? 'dark' : 'light'}
        </button>
      </div>
    </div>
  );
};

export default AnalogClock;