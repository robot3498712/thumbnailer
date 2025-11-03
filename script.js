/**
 * thumbnailer :: web app
 */

document.addEventListener("DOMContentLoaded", ev => {

	const dWidth = document.body.dataset.width;
	const dFit = document.body.dataset.fit == "true";
	const maxHeight = window.innerHeight * 0.85;

    const images = document.querySelectorAll('ul.flex li img');
	const lightbox = document.getElementById('lightbox');
	const lightboxImage = document.getElementById('lightboxImage');
	const lightboxImageTransition = lightboxImage.style.transition;
	const lightboxLoading = document.getElementById('lightboxLoading');
	const lightboxClose = document.getElementById('lightboxClose');

	// transformers
	let rot = [];
	let scale = 1;
	const scaleStep = 0.1;
	const minScale = 0.5;
	const maxScale = 5;

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
				container.style.height = `${maxHeight}px`;
				container.style.overflow = 'hidden';
				container.style.position = 'relative';

				let fade = document.createElement('div');
				fade.className = 'fadeout';
				Object.assign(fade.style, {
					position: 'absolute',
					left: 0,
					right: 0,
					bottom: 0,
					height: '50%',
					background: 'linear-gradient(to bottom, rgba(0,0,0,0), rgba(0,0,0,1))',
					mixBlendMode: 'destination-out',
					pointerEvents: 'none'
				});
				container.appendChild(fade);
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

	const openLightbox = (imageSrc, imageID) => {
		const img = new Image();
		img.onload = () => {
			transform(true, imageID);

			void lightboxImage.offsetHeight; // force reflow before restore
			lightboxImage.style.transition = lightboxImageTransition;

			lightboxLoading.style.display = "none";
			lightboxImage.src = img.src;
			lightboxImage.dataset.id = imageID;
			lightbox.style.display = 'flex';
		};
		img.onerror = () => {
			// implements a one-time retry, forcing server-side decode
			if (lightboxImage.dataset.id == imageID) return;

			fetch(`/image/${imageID}?retry=1`)
			.then(response => {
				lightboxImage.dataset.id = imageID;
				if (!response.ok) throw new Error('Failed to load image');
				return response.blob();
			})
			.then(imageBlob => {
				openLightbox(URL.createObjectURL(imageBlob), imageID);
			})
			.catch(error => {
				console.error("Fetch image -> Error:", error);
			});
		};
		img.src = imageSrc;
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

			lightboxImage.src = "";
			lightboxLoading.style.display = "block";
			lightbox.style.display = 'flex';

			const imageID = img.getAttribute('data-id');
			fetch(`/image/${imageID}`)
			.then(response => {
				if (!response.ok) throw new Error('Failed to load image');
				return response.blob();
			})
			.then(imageBlob => {
				openLightbox(URL.createObjectURL(imageBlob), imageID);
			})
			.catch(error => {
				lightbox.style.display = 'none';
				console.error("Fetch image -> Error:", error);
			});
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
			const nextImageID = nextListItem.querySelector("img").getAttribute("data-id");
			const fullImageUrl = '/image/' + nextImageID;

			lightboxImage.src = "";
			lightboxLoading.style.display = "block";

			fetch(fullImageUrl)
			.then(response => response.blob())
			.then(imageBlob => {
				openLightbox(URL.createObjectURL(imageBlob), nextImageID);
			});
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
