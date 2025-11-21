# thumbnailer cli

## [0.2.1] (2025-11-21)

Localhosted web app for displaying thumbnails in a grid layout. 

Using https://github.com/libvips/libvips with https://github.com/cshum/vipsgen 
and https://github.com/bfabiszewski/libmobi for Mobipocket/Kindle


### Supported media formats

* jpeg, png, gif, webp, bmp, heic, avif, svg, tiff, jp2, jxl
* pdf, epub, mobi, azw3
* [untested] azw, azw4, pdb, prc


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

### Future plans, pending features & issues

* refactor, especially conditional processing

* search / index via menu
	* by extension / format

* support for djvu, raw image

* caching flag
	* for example using sqlite

* add more filetypes
	* https://github.com/h2non/filetype.py

* consider config defaults for ip:port and such

* color scheme; day/night mode


### License

thumbnailer is free software: you can redistribute it and/or modify it under the terms of the GNU General Public License as published by the Free Software Foundation, either version 3 of the License, or (at your option) any later version.
thumbnailer is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU General Public License for more details.
You should have received a copy of the GNU General Public License along with thumbnailer. If not, see <http://www.gnu.org/licenses/>
