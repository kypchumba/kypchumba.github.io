window.addEventListener("DOMContentLoaded", () => {
  const revealItems = document.querySelectorAll(".reveal");
  const typingRole = document.querySelector(".typing-role");
  const toggleProjectsButton = document.querySelector(".toggle-projects");
  const extraProjects = document.querySelector("#more-projects");
  const contactForm = document.querySelector(".contact-form");

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

  if (typingRole) {
    const roles = JSON.parse(typingRole.dataset.roles || "[]");
    let roleIndex = 0;
    let charIndex = 0;
    let isDeleting = false;

    const typeNextRole = () => {
      const currentRole = roles[roleIndex] || "";

      typingRole.textContent = currentRole.slice(0, charIndex);

      let delay = isDeleting ? 65 : 95;

      if (!isDeleting && charIndex === currentRole.length) {
        delay = 1200;
        isDeleting = true;
      } else if (isDeleting && charIndex === 0) {
        isDeleting = false;
        roleIndex = (roleIndex + 1) % roles.length;
        delay = 240;
      } else {
        charIndex += isDeleting ? -1 : 1;
      }

      window.setTimeout(typeNextRole, delay);
    };

    if (roles.length > 0) {
      typeNextRole();
    }
  }

  if (toggleProjectsButton && extraProjects) {
    extraProjects.hidden = true;

    toggleProjectsButton.addEventListener("click", () => {
      const willShow = extraProjects.hidden;

      extraProjects.hidden = !willShow;
      toggleProjectsButton.textContent = willShow
        ? "Hide More Projects"
        : "View More Projects";
      toggleProjectsButton.setAttribute(
        "aria-expanded",
        willShow ? "true" : "false"
      );
    });
  }

  if (contactForm) {
    contactForm.addEventListener("submit", (event) => {
      event.preventDefault();

      const submitButton = contactForm.querySelector(".submit-button");

      if (!submitButton) {
        return;
      }

      const originalText = submitButton.textContent;
      submitButton.textContent = "Message Sent";
      submitButton.disabled = true;

      window.setTimeout(() => {
        contactForm.reset();
        submitButton.textContent = originalText;
        submitButton.disabled = false;
      }, 1800);
    });
  }
});
