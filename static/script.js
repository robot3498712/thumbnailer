/**
 * thumbnailer :: web app
 */

document.addEventListener("DOMContentLoaded", ev => {
	const scheme = localStorage.getItem("scheme");
	if (scheme) {
		if (scheme == "light") document.body.classList.add("light");
		else document.body.classList.remove("light");
	} else {
		if (window.matchMedia("(prefers-color-scheme: light)").matches) document.body.classList.add("light");
		else document.body.classList.remove("light");
	}

	const dWidth = document.body.dataset.width;
	const dFit = document.body.dataset.fit == "true";
	const maxHeight = window.innerHeight * 0.85;

    const images = document.querySelectorAll('ul.flex li img');
	const lightbox = document.getElementById('lightbox');
	const lightboxImage = document.getElementById('lightboxImage');
	const lightboxClose = document.getElementById('lightboxClose');

	// touch device
	let lastTouchX = 0;
	let lastTouchY = 0;
	let isPanning = false;
	let pinchStartDist = 0;
	let touchEndX = 0;
	const swipeThreshold = 100;

	// transformers
	let rot = [];
	let scale = 1;
	const scaleStep = 0.33;
	const minScale = 1.00;
	const maxScale = 10;

	let translateX = 0;
	let translateY = 0;
	let lastMouseX = 0;
	let lastMouseY = 0;

	const tfGetDegrees = (deg, dir) => {
		if (dir == "r") deg += 90; 
		else if (dir == "l") deg = (deg == 0 ? 360 : deg) - 90;
		return (deg % 360 + 360) % 360;
	};

	const transform = (ro=false, imageID=null) => {
		if (ro) {
			scale = 1;
			translateX = 0;
			translateY = 0;
			lightboxImage.style.transition = "none";
			lightboxImage.style.transform = "";

			if (rot.hasOwnProperty(imageID)) lightboxImage.style.transform = `rotate(${rot[imageID]}deg)`;
			else lightboxImage.style.transform = "";
			return;
		}

		// auto-recenter if image can be fully visible
		const rect = lightboxImage.getBoundingClientRect();
		if (rect.width <= window.innerWidth && rect.height <= window.innerHeight) {
			translateX = 0;
			translateY = 0;
		}

		let tfStr = `scale(${scale}) translate(${translateX}px, ${translateY}px)`;
		if (imageID && rot.hasOwnProperty(imageID)) tfStr = `rotate(${rot[imageID]}deg) ${tfStr}`;

		lightboxImage.style.transform = tfStr;
	};

	// lazy-loading
	const loadImage = (entry, observer) => {
		const img = entry.target;
		const container = img.parentNode;
		img.src = `/thumbnail/${img.dataset.id}`;

		img.onload = () => {
			if (img.naturalWidth * 3 < dWidth) { // pixelartify vsmall images
				img.style.imageRendering = 'crisp-edges';
			}
			img.style.opacity = 1;
			container.style.minHeight = '0px';

			// implements vertical crop with fadeout
			// img.naturalHeight may be incorrect for some payloads (like avif source)
			if (dFit && img.getBoundingClientRect().height > maxHeight) {
				Object.assign(container.style, {
					height: `${maxHeight}px`,
					overflow: 'hidden',
					position: 'relative',
					WebkitMaskImage: 'linear-gradient(to bottom, black 50%, transparent 100%)',
					maskImage: 'linear-gradient(to bottom, black 50%, transparent 100%)'
				});
			}
		}
		observer.unobserve(entry.target);
	};

	const observer = new IntersectionObserver((entries, observer) => {
		entries.forEach(entry => {
			if (entry.isIntersecting) {
				loadImage(entry, observer);
			}
		});
	}, {
		rootMargin: '100px 0px', // load before the image reaches the viewport
		threshold: 0.1           // trigger when 10% of the image is in the viewport
	});

	const openLightbox = (imageID) => {
		const showNewImage = () => {
			lightboxImage.style.transition = 'none';
			lightboxImage.style.opacity = 0;

			lightboxImage.removeAttribute('src');
			lightboxImage.dataset.id = imageID;

			lightbox.style.display = 'flex';
			transform(true, imageID);

			void lightboxImage.offsetWidth; // force re-flow
			lightboxImage.src = `/image/${imageID}`;

			requestAnimationFrame(() => {
				lightboxImage.style.transition = "";
				lightboxImage.style.opacity = 1;
			});
		};

		if (lightbox.style.display !== 'flex' || window.getComputedStyle(lightboxImage).opacity == 0) {
			showNewImage();
			return;
		}
		lightboxImage.style.opacity = 0;

		lightboxImage.addEventListener('transitionend', function handler() {
			lightboxImage.removeEventListener('transitionend', handler);
			showNewImage();
		}, { once: true });
	};

	lightboxImage.onload = () => {
		void lightboxImage.offsetHeight; // force re-flow
	};

	lightboxImage.onerror = () => {
		// implements a one-time retry, forcing server-side decode
		if (lightboxImage.src.includes("retry=1")) {
			return;
		}
		lightboxImage.src = `/image/${lightboxImage.dataset.id}?retry=1`;
	};

	// attach handlers
	// lightbox -> new tab
	document.getElementById("lightboxImage").addEventListener("click", e => {
		window.open(e.target.src, "_blank");
		lightbox.style.display = 'none';
	});

    images.forEach(img => {
		// lazy-loading
		observer.observe(img);

		// lightbox
		img.parentNode.addEventListener('click', ev => {
			ev.preventDefault();
			openLightbox(img.getAttribute('data-id'));
		});

		// right-click
        img.addEventListener("contextmenu", ev => {
			ev.preventDefault();
            fetch("/context/", {
                method: "POST",
                headers: {"Content-Type": "application/json"},
                body: JSON.stringify({id: parseInt(img.dataset.id, 10)})
            })
            .then(response => response.json())
            .catch(error => {
                console.error("Context open -> Error:", error);
            });
        });
    });

	lightboxClose.addEventListener('click', () => {
		lightbox.style.display = 'none';
	});

	lightbox.addEventListener('click', ev => {
		if (ev.target === lightbox) lightbox.style.display = 'none';
	});

	// lighbox keys
	document.addEventListener("keydown", ev => {
		if (lightbox.style.display !== 'flex') return;

		const el = document.querySelector(`ul.flex li img[data-id='${lightboxImage.dataset.id}']`).parentElement;
		if (!el) return;

		let nextListItem;

		switch (true) {
			case ev.key === "ArrowRight":
				ev.preventDefault();
				nextListItem = el.nextElementSibling;
				break;
			case ev.key === "ArrowLeft":
				ev.preventDefault();
				nextListItem = el.previousElementSibling;
				break;
			case ev.key === "l":
			case ev.key === "r":
				ev.preventDefault();
				let deg = tfGetDegrees(rot[lightboxImage.dataset.id] ? rot[lightboxImage.dataset.id] : 0, ev.key);
				rot[lightboxImage.dataset.id] = deg;
				transform(true, lightboxImage.dataset.id);
				break;
			case /^F\d{1,2}$/.test(ev.key):
				break;
			case ev.key === "+":
				ev.preventDefault();
				scale = Math.min(scale + scaleStep, maxScale);
				transform(false, lightboxImage.dataset.id);
				break;
			case ev.key === "-":
				ev.preventDefault();
				scale = Math.max(scale - scaleStep, minScale);
				transform(false, lightboxImage.dataset.id);
				break;
			default:
				ev.preventDefault();
				lightbox.style.display = 'none';
				return;
		}

		if (nextListItem) {
			openLightbox(nextListItem.querySelector("img").getAttribute("data-id"));
		}
	});

	// zoom
	lightbox.addEventListener("wheel", ev => {
		ev.preventDefault();
		if (ev.deltaY < 0) {
			scale = Math.min(scale + scaleStep, maxScale);
		} else {
			scale = Math.max(scale - scaleStep, minScale);
		}
		transform(false, lightboxImage.dataset.id);
	});

	// touch device
	lightbox.addEventListener("touchstart", ev => {
		if (lightbox.style.display !== "flex") return;

		if (ev.touches.length === 1) {
			if (scale > 1) {
				isPanning = true;
				lastTouchX = ev.touches[0].clientX;
				lastTouchY = ev.touches[0].clientY;
			} else { // swipe nav
				lastTouchX = ev.touches[0].screenX;
			}
		} else {
			isPanning = false; // multi-touch
		}
	});

	lightbox.addEventListener("touchmove", ev => {
		if (lightbox.style.display !== "flex") return;
		ev.preventDefault();

		if (ev.touches.length === 1 && isPanning) {
			ev.preventDefault();
			const touch = ev.touches[0];
			const dx = touch.clientX - lastTouchX;
			const dy = touch.clientY - lastTouchY;

			translateX += dx;
			translateY += dy;

			transform(false, lightboxImage.dataset.id);
			lastTouchX = touch.clientX;
			lastTouchY = touch.clientY;

		} else if (ev.touches.length === 2) { // pinch zoom
			ev.preventDefault();
			const [t1, t2] = ev.touches;
			const dx = t2.clientX - t1.clientX;
			const dy = t2.clientY - t1.clientY;
			const dist = Math.sqrt(dx * dx + dy * dy);

			if (!pinchStartDist) {
				pinchStartDist = dist;
				return;
			}

			const scaleChange = dist / pinchStartDist;
			scale = Math.min(Math.max(scale * scaleChange, minScale), maxScale);

			transform(false, lightboxImage.dataset.id);
			pinchStartDist = dist;
		}
	}, { passive: false });

	lightbox.addEventListener("touchend", ev => {
		if (scale <= 1 && ev.changedTouches.length === 1) { // swipe nav
			const touchEndX = ev.changedTouches[0].screenX;
			const diffX = touchEndX - lastTouchX;
			if (Math.abs(diffX) >= swipeThreshold) {
				const key = diffX < 0 ? "ArrowRight" : "ArrowLeft";
				document.dispatchEvent(new KeyboardEvent("keydown", { key }));
			}
		}
		// reset pan & pinch states
		if (ev.touches.length === 0) {
			isPanning = false;
			pinchStartDist = 0;
		} else if (ev.touches.length === 1 && scale > 1) {
			lastTouchX = ev.touches[0].clientX;
			lastTouchY = ev.touches[0].clientY;
			isPanning = true;
		}
	});

	// pan (with rotation support)
	lightbox.addEventListener("mousemove", e => {
		if (lightbox.style.display !== "flex") return;
		if (!lightboxImage.src) return;

		const rect = lightboxImage.getBoundingClientRect();
		if (rect.width <= window.innerWidth && rect.height <= window.innerHeight) return;

		if (lastMouseX && lastMouseY) {
			const dx = e.clientX - lastMouseX;
			const dy = e.clientY - lastMouseY;

			const deg = rot[lightboxImage.dataset.id] || 0;
			const rad = deg * Math.PI / 180;
			const cos = Math.cos(rad);
			const sin = Math.sin(rad);

			// apply rotation transform to movement
			const dxRot = dx * cos + dy * sin;
			const dyRot = -dx * sin + dy * cos;

			translateX -= dxRot;
			translateY -= dyRot;

			// containment correction for 90° / 270°
			let maxX, maxY;
			if (Math.abs((deg % 180 + 180) % 180 - 90) < 1e-3) {
				// rotated 90° or 270° -> swap width/height limits
				maxX = Math.max(0, (rect.height - window.innerWidth) / 2);
				maxY = Math.max(0, (rect.width - window.innerHeight) / 2);
			} else {
				maxX = Math.max(0, (rect.width - window.innerWidth) / 2);
				maxY = Math.max(0, (rect.height - window.innerHeight) / 2);
			}

			translateX = Math.max(-maxX, Math.min(maxX, translateX));
			translateY = Math.max(-maxY, Math.min(maxY, translateY));

			transform(false, lightboxImage.dataset.id);
		}

		lastMouseX = e.clientX;
		lastMouseY = e.clientY;
	});

	lightbox.addEventListener("mouseleave", () => {
		lastMouseX = 0;
		lastMouseY = 0;
	});

	document.getElementById("btn-mode").addEventListener("click", e => {
		let result = document.body.classList.toggle("light");
		localStorage.scheme = result ? "light" : "dark";
	});

	// menu
	document.getElementById('menu').addEventListener('click', () => {
		const menuList = document.getElementById('menuList');
		if (menuList.style.display === 'block') {
			menuList.style.display = 'none';
			document.getElementById('menuOverlay').style.display = 'none';
		} else {
			menuList.style.display = 'block';
			document.getElementById('menuOverlay').style.display = 'block';
		}
	});

	(() => {
		document.querySelectorAll('.dir-container').forEach(container => {
			const listItem = document.createElement('li');
			listItem.textContent = container.querySelector('span').textContent;
			listItem.setAttribute('data-target', container.id);
			listItem.addEventListener('click', (event) => {
				menuList.style.display = 'none';
				document.getElementById('menuOverlay').style.display = 'none';
				document.getElementById(event.target.getAttribute('data-target')).scrollIntoView();
			});
			document.getElementById('menuList').appendChild(listItem);
		});
	})();
});
