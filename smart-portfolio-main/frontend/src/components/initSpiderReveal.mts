export function initSpiderReveal() {
  const trigger = document.getElementById("profile-trigger");
  const revealContainer = document.getElementById("reveal-container");
  const nameDisplay = document.getElementById("name-display");
  const customCursor = document.getElementById("custom-cursor");
  const ProfilePic = document.getElementById("profile-img-base");
  const nameOriginal = document.getElementById("name-original");
  const nameSpider = document.getElementById("name-spider");

  if (!trigger ||
    !revealContainer ||
    !nameDisplay ||
    !customCursor ||
    !ProfilePic ||
    !nameOriginal ||
    !nameSpider)
    return;

  const handleMove = (e: MouseEvent) => {
    // 1. Move the visual custom cursor (Global coordinates)
    customCursor.style.left = `${e.clientX}px`;
    customCursor.style.top = `${e.clientY}px`;

    // 2. Update the Reveal Mask (Local coordinates)
    const rect = trigger.getBoundingClientRect();
    const x = e.clientX - rect.left;
    const y = e.clientY - rect.top;

    // Update the circle mask size and position
    // 80px is the radius of the "flashlight"
    revealContainer.style.clipPath = `circle(100px at ${x}px ${y}px)`;
  };

  trigger.addEventListener("mouseenter", () => {
    customCursor.style.opacity = "1";
    nameOriginal.style.opacity = "0";
    nameSpider.style.opacity = "1";
    nameDisplay.classList.replace("text-amber-400", "text-red-600");
    revealContainer.style.clipPath = `circle(0% at 50% 50%)`;
    document.body.style.cursor = "none";
  });

  trigger.addEventListener("mousemove", handleMove);

  trigger.addEventListener("mouseleave", () => {
    customCursor.style.opacity = "0";
    nameOriginal.style.opacity = "1";
    nameSpider.style.opacity = "0";
    nameDisplay.classList.replace("text-red-600", "text-amber-400");
    document.body.style.cursor = "default";
  });
}
