/**
 * thumbnailer web app
 */

function openLightbox(imageSrc, imageID) {

    const img = new Image();
	img.onload = function() {
		lightboxLoading.style.display = "none";
		lightboxImage.src = img.src;
		lightboxImage.dataset.id = imageID;
		lightbox.style.display = 'flex';
	};
	img.onerror = function() {
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
}

function makeMenu() {
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
}

document.addEventListener("DOMContentLoaded", function(ev) {

	const dWidth = document.body.dataset.width;
    const images = document.querySelectorAll('ul.flex li img');
	const lightbox = document.getElementById('lightbox');
	const lightboxImage = document.getElementById('lightboxImage');
	const lightboxLoading = document.getElementById('lightboxLoading');
	const lightboxClose = document.getElementById('lightboxClose');

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
		}
		observer.unobserve(entry.target);
	};

	const options = {
		rootMargin: '100px 0px', // load before the image reaches the viewport
		threshold: 0.1           // trigger when 10% of the image is in the viewport
	};

	const observer = new IntersectionObserver((entries, observer) => {
		entries.forEach(entry => {
			if (entry.isIntersecting) {
				loadImage(entry, observer);
			}
		});
	}, options);

	// attach handlers
	// lightbox -> new tab
	document.getElementById("lightboxImage").addEventListener("click", function() {
		window.open(this.src, "_blank");
		lightbox.style.display = 'none';
	});

    images.forEach(function(img) {
		// lazy-loading
		observer.observe(img);

		// lightbox
		img.parentNode.addEventListener('click', function(ev) {
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
        img.addEventListener("contextmenu", function(ev) {
			ev.preventDefault();
            fetch("/context/", {
                method: "POST",
                headers: {"Content-Type": "application/json"},
                body: JSON.stringify({id: parseInt(img.dataset.id, 10)})
            })
            .then(response => response.json())
            /*.then(data => {
                console.log("Context open:", data);
            })*/
            .catch(error => {
                console.error("Context open -> Error:", error);
            });
        });
    });

	lightboxClose.addEventListener('click', function() {
		lightbox.style.display = 'none';
	});

	lightbox.addEventListener('click', function(ev) {
		if (ev.target === lightbox) lightbox.style.display = 'none';
	});

	// Lighbox keys
	document.addEventListener("keydown", function(ev) {
		if (lightbox.style.display !== 'flex') return;
		ev.preventDefault();

		const el = document.querySelector(`ul.flex li img[data-id='${lightboxImage.dataset.id}']`).parentElement;
		if (!el) return;

		let nextListItem;

		switch (true) {
			case ev.key === "ArrowRight":
				nextListItem = el.nextElementSibling;
				break;
			case ev.key === "ArrowLeft":
				nextListItem = el.previousElementSibling;
				break;
			default:
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

	// Menu setup
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
	makeMenu();
});
