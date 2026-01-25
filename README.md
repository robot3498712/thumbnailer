# thumbnailer cli

## [0.2.4] (2026-01-24)

Localhosted web app for displaying thumbnails in a grid layout. 

Using https://github.com/libvips/libvips [8.18] with https://github.com/cshum/vipsgen 
and https://github.com/bfabiszewski/libmobi for Mobipocket/Kindle


### Supported media formats

* jpeg, png, gif, webp, bmp, heic, avif, svg, tiff, jp2, jxl
* pdf, epub, mobi, azw3
* [untested] azw, azw4, pdb, prc
* [slow] raw image formats


### Module notes

ImageMagick (vips-magick < libMagickCore) is required for formats without a dedicated loader (such as bmp). 
Refer to https://github.com/libvips/libvips/blob/master/libvips/foreign/magickload.c


### Manual

```
./thumbnailer [<flags>] <root path containing media>
	-h for help
```

* Refer to server info printed and open link in browser

* Click thumbnail to open lightbox

#### Key bindings
##### Lightbox
```
<-		previous

->		next

l		rotate left

r		rotate right

+/wheel	zoom in

-/wheel	zoom out

any		exit lightbox

mouseLeft	open image in new tab
```
##### Grid
```
mouseRight	open file in native app
```

#### Image Presets

```
none	the default, no resizing
hd		target 1080p (~2 MP)
4k		target 2160p (~8 MP)

There is a performance trade-off, albeit for non-local client (mobile device) a recode might serve more efficiently.
```


### Future plans, pending features & issues

* refactor, especially conditional processing
	* see https://github.com/libvips/libvips/blob/master/libvips/foreign/dcrawload.c#L57 and other loaders for suffix enumerations

* search / index via menu
	* by extension / format

* support for djvu

* caching flag
	* for example using sqlite

* add more filetypes
	* https://github.com/h2non/filetype.py

* consider config defaults for ip:port and such

* raw image handling
	* Most time is spent in dcraw, so performance isn't that much better than the previous solution with imagemagick.
	* https://www.libvips.org/2025/12/04/What's-new-in-8.18.html

* ui option, such as luart [ in a custom repo ]
	* ideally bi-directional IPC with dynamic updates


### License

thumbnailer is free software: you can redistribute it and/or modify it under the terms of the GNU General Public License as published by the Free Software Foundation, either version 3 of the License, or (at your option) any later version.
thumbnailer is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU General Public License for more details.
You should have received a copy of the GNU General Public License along with thumbnailer. If not, see <http://www.gnu.org/licenses/>
