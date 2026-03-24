const revealItems = document.querySelectorAll(".reveal");
const contactForm = document.querySelector(".contact-form");
const counters = document.querySelectorAll(".count-up");
const typeRole = document.querySelector(".type-role");
const navLinks = [...document.querySelectorAll(".site-nav a")];
const nav = document.querySelector(".site-nav");
const navIndicator = document.querySelector(".nav-indicator");

if ("IntersectionObserver" in window) {
  const observer = new IntersectionObserver(
    (entries) => {
      entries.forEach((entry) => {
        if (entry.isIntersecting) {
          entry.target.classList.add("is-visible");
          observer.unobserve(entry.target);
        }
      });
    },
    { threshold: 0.18 }
  );

  revealItems.forEach((item) => observer.observe(item));
} else {
  revealItems.forEach((item) => item.classList.add("is-visible"));
}

if (contactForm) {
  contactForm.addEventListener("submit", (event) => {
    event.preventDefault();

    const button = contactForm.querySelector("button");
    if (!button) {
      return;
    }

    const originalText = button.textContent;
    button.textContent = "Message Sent";
    button.disabled = true;

    window.setTimeout(() => {
      contactForm.reset();
      button.textContent = originalText;
      button.disabled = false;
    }, 1800);
  });
}

if (typeRole) {
  const roles = JSON.parse(typeRole.dataset.roles || "[]");
  let roleIndex = 0;
  let charIndex = roles[0]?.length || 0;
  let isDeleting = false;

  const runTypeCycle = () => {
    const currentRole = roles[roleIndex] || "";

    typeRole.textContent = currentRole.slice(0, charIndex);

    let delay = isDeleting ? 70 : 100;

    if (!isDeleting && charIndex === currentRole.length) {
      delay = 1000;
      isDeleting = true;
    } else if (isDeleting && charIndex === 0) {
      isDeleting = false;
      roleIndex = (roleIndex + 1) % roles.length;
      delay = 260;
    } else {
      charIndex += isDeleting ? -1 : 1;
    }

    window.setTimeout(runTypeCycle, delay);
  };

  window.setTimeout(runTypeCycle, 1000);
}

const formatCount = (value, target, suffix = "") => {
  const isWhole = Number.isInteger(target);
  const display = isWhole ? Math.round(value).toString() : value.toFixed(1);
  return `${display}${suffix}`;
};

const animateCounter = (counter) => {
  if (counter.dataset.animated === "true") {
    return;
  }

  counter.dataset.animated = "true";

  const target = Number(counter.dataset.target || "0");
  const suffix = counter.dataset.suffix || "";
  const duration = 1600;
  const startTime = performance.now();

  const tick = (now) => {
    const progress = Math.min((now - startTime) / duration, 1);
    const eased = 1 - Math.pow(1 - progress, 3);
    const current = target * eased;

    counter.textContent = formatCount(current, target, suffix);

    if (progress < 1) {
      window.requestAnimationFrame(tick);
    } else {
      counter.textContent = formatCount(target, target, suffix);
    }
  };

  window.requestAnimationFrame(tick);
};

if ("IntersectionObserver" in window) {
  const counterObserver = new IntersectionObserver(
    (entries) => {
      entries.forEach((entry) => {
        if (entry.isIntersecting) {
          animateCounter(entry.target);
          counterObserver.unobserve(entry.target);
        }
      });
    },
    { threshold: 0.45 }
  );

  counters.forEach((counter) => counterObserver.observe(counter));
} else {
  counters.forEach((counter) => animateCounter(counter));
}

const navItems = navLinks
  .map((link) => {
    const target = document.querySelector(link.getAttribute("href"));
    if (!target) {
      return null;
    }

    return { link, target };
  })
  .filter(Boolean);

const setIndicator = (link) => {
  if (!nav || !navIndicator || !link) {
    return;
  }

  nav.style.setProperty("--indicator-x", `${link.offsetLeft}px`);
  nav.style.setProperty("--indicator-width", `${link.offsetWidth}px`);
  nav.style.setProperty("--indicator-opacity", "1");
};

const setActiveLink = (activeLink) => {
  navLinks.forEach((link) => {
    link.classList.toggle("active", link === activeLink);
  });

  setIndicator(activeLink);
};

const getCurrentSection = () => {
  const headerOffset = 140;
  const scrollPoint = window.scrollY + headerOffset;

  let current = navItems[0];

  navItems.forEach((item) => {
    if (item.target.offsetTop <= scrollPoint) {
      current = item;
    }
  });

  return current;
};

let ticking = false;

const updateActiveNav = () => {
  const current = getCurrentSection();

  if (current) {
    setActiveLink(current.link);
  }

  ticking = false;
};

const requestNavUpdate = () => {
  if (ticking) {
    return;
  }

  ticking = true;
  window.requestAnimationFrame(updateActiveNav);
};

window.addEventListener("scroll", requestNavUpdate, { passive: true });
window.addEventListener("resize", requestNavUpdate);
window.addEventListener("load", requestNavUpdate);

navLinks.forEach((link) => {
  link.addEventListener("click", () => {
    window.setTimeout(requestNavUpdate, 80);
  });
});

requestNavUpdate();
