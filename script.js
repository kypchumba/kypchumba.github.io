window.addEventListener("DOMContentLoaded", () => {
  const revealItems = document.querySelectorAll(".reveal");
  const typingRole = document.querySelector(".typing-role");
  const toggleProjectsButton = document.querySelector(".toggle-projects");
  const extraProjects = document.querySelector("#more-projects");
  const contactForm = document.querySelector(".contact-form");
  const projectCards = document.querySelectorAll(".project-card");
  const projectModal = document.querySelector("#project-modal");
  const projectModalClose = document.querySelector(".project-modal-close");
  const projectModalCloseInline = document.querySelector(".project-modal-close-inline");
  const projectModalDialog = document.querySelector(".project-modal-dialog");
  const projectModalTitle = document.querySelector("#project-modal-title");
  const projectModalImage = document.querySelector("#project-modal-image");
  const projectModalLinks = document.querySelector("#project-modal-links");
  const projectModalLanguages = document.querySelector("#project-modal-languages");
  const projectModalDescription = document.querySelector("#project-modal-description");
  const contactApiMeta = document.querySelector('meta[name="contact-api-url"]');
  const contactApiURL =
    contactApiMeta?.getAttribute("content")?.trim() || `${window.location.origin}/contact`;
  const modalTransitionDuration = 260;
  let activeProjectCard = null;
  let modalCloseTimer = null;

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

  if (
    projectCards.length > 0 &&
    projectModal &&
    projectModalClose &&
    projectModalCloseInline &&
    projectModalDialog &&
    projectModalTitle &&
    projectModalImage &&
    projectModalLinks &&
    projectModalLanguages &&
    projectModalDescription
  ) {
    const updateModalOverflow = () => {
      window.requestAnimationFrame(() => {
        const hasOverflow =
          projectModalDialog.scrollHeight > projectModalDialog.clientHeight + 2;

        projectModalDialog.classList.toggle("can-scroll", hasOverflow);
      });
    };

    const closeProjectModal = () => {
      projectModal.classList.remove("is-open");
      document.body.classList.remove("modal-open");

      if (modalCloseTimer) {
        window.clearTimeout(modalCloseTimer);
      }

      modalCloseTimer = window.setTimeout(() => {
        projectModal.hidden = true;
        projectModal.setAttribute("aria-hidden", "true");
        projectModalLinks.innerHTML = "";
        projectModalLanguages.innerHTML = "";
        projectModalImage.src = "";
        projectModalImage.alt = "";
        projectModalDialog.classList.remove("can-scroll");
        projectModalDialog.scrollTop = 0;
        modalCloseTimer = null;
      }, modalTransitionDuration);

      if (activeProjectCard) {
        activeProjectCard.focus();
        activeProjectCard = null;
      }
    };

    const openProjectModal = (projectCard) => {
      const image = projectCard.querySelector(".project-screenshot img");
      const title = projectCard.querySelector(".project-body h3");
      const description = projectCard.querySelector(".project-summary");
      const codeUrl = projectCard.dataset.codeUrl;
      const liveUrl = projectCard.dataset.liveUrl;
      const languages = projectCard.dataset.languages || "Not specified";
      const fullDescription =
        projectCard.dataset.fullDescription || description?.textContent?.trim() || "";

      if (!image || !title || !description) {
        return;
      }

      if (modalCloseTimer) {
        window.clearTimeout(modalCloseTimer);
        modalCloseTimer = null;
      }

      projectModalTitle.textContent = title.textContent.trim();
      projectModalImage.src = image.src;
      projectModalImage.alt = image.alt;
      projectModalDescription.textContent = fullDescription;
      projectModalLinks.innerHTML = "";
      projectModalLanguages.innerHTML = "";

      if (codeUrl) {
        const codeLink = document.createElement("a");
        const codeIcon = document.createElement("img");
        codeLink.href = codeUrl;
        codeLink.target = "_blank";
        codeLink.rel = "noopener noreferrer";
        codeLink.className = "project-modal-link project-modal-link-code";
        codeLink.setAttribute("aria-label", "GitHub Code");
        codeIcon.src = "assets/socialmedia/github.png";
        codeIcon.alt = "GitHub";
        codeLink.appendChild(codeIcon);
        projectModalLinks.appendChild(codeLink);
      }

      if (liveUrl) {
        const liveLink = document.createElement("a");
        const liveIcon = document.createElement("img");
        liveLink.href = liveUrl;
        liveLink.target = "_blank";
        liveLink.rel = "noopener noreferrer";
        liveLink.className = "project-modal-link project-modal-link-live";
        liveLink.setAttribute("aria-label", "Live Preview");
        liveIcon.src = "assets/socialmedia/code.png";
        liveIcon.alt = "Live Preview";
        liveLink.appendChild(liveIcon);
        projectModalLinks.appendChild(liveLink);
      }

      languages
        .split(",")
        .map((language) => language.trim())
        .filter(Boolean)
        .forEach((language) => {
          const chip = document.createElement("span");
          chip.className = "project-language-chip";
          chip.textContent = language;
          projectModalLanguages.appendChild(chip);
        });

      activeProjectCard = projectCard;
      projectModal.hidden = false;
      projectModal.setAttribute("aria-hidden", "false");
      document.body.classList.add("modal-open");
      window.requestAnimationFrame(() => {
        projectModal.classList.add("is-open");
        updateModalOverflow();
      });
      projectModalCloseInline.focus();
    };

    projectCards.forEach((projectCard) => {
      projectCard.addEventListener("click", (event) => {
        openProjectModal(projectCard);
      });

      projectCard.addEventListener("keydown", (event) => {
        if (event.key === "Enter" || event.key === " ") {
          event.preventDefault();
          openProjectModal(projectCard);
        }
      });
    });

    projectModal.addEventListener("click", (event) => {
      if (event.target.closest("[data-close-modal='true']")) {
        closeProjectModal();
      }
    });

    projectModalClose.addEventListener("click", closeProjectModal);
    projectModalCloseInline.addEventListener("click", closeProjectModal);

    document.addEventListener("keydown", (event) => {
      if (event.key === "Escape" && !projectModal.hidden) {
        closeProjectModal();
      }
    });

    window.addEventListener("resize", () => {
      if (!projectModal.hidden) {
        updateModalOverflow();
      }
    });
  }

  if (contactForm) {
    contactForm.addEventListener("submit", (event) => {
      event.preventDefault();

      const submitButton = contactForm.querySelector(".submit-button");
      const feedback = contactForm.querySelector(".contact-feedback");
      const formData = new FormData(contactForm);
      const payload = {
        name: String(formData.get("name") || "").trim(),
        email: String(formData.get("email") || "").trim(),
        message: String(formData.get("message") || "").trim(),
      };

      if (!submitButton) {
        return;
      }

      if (feedback) {
        feedback.textContent = "";
        feedback.classList.remove("is-error", "is-success");
      }

      if (!payload.name || !payload.email || !payload.message) {
        if (feedback) {
          feedback.textContent = "Please fill in your name, email, and message.";
          feedback.classList.add("is-error");
        }
        return;
      }

      submitButton.disabled = true;
      submitButton.querySelector(".text").textContent = "Sending...";

      fetch(contactApiURL, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(payload),
      })
        .then(async (response) => {
          const result = await response.json().catch(() => ({}));

          if (!response.ok) {
            throw new Error(result.error || "Unable to send your message right now.");
          }

          contactForm.reset();
          if (feedback) {
            feedback.textContent = result.message || "Message sent successfully.";
            feedback.classList.add("is-success");
          }
        })
        .catch((error) => {
          if (feedback) {
            feedback.textContent = error.message;
            feedback.classList.add("is-error");
          }
        })
        .finally(() => {
          submitButton.disabled = false;
          submitButton.querySelector(".text").textContent = "Send Message";
        });
    });
  }
});
