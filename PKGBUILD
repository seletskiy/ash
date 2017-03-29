pkgname=ash
pkgver=20160715
pkgrel=1
pkgdesc="Atlassian Stash reviews in editor"
arch=('i686' 'x86_64')
license=('GPL')
makedepends=('go' 'git')

source=(
	"ash::git://github.com/seletskiy/ash.git"
)

md5sums=(
	'SKIP'
)

backup=(
)

pkgver() {
	cd "$srcdir/$pkgname"
	git log -1 --format="%cd" --date=short | sed s/-//g
}

build() {
	cd "$srcdir/$pkgname"

	if [ -L "$srcdir/$pkgname" ]; then
		rm "$srcdir/$pkgname" -rf
		mv "$srcdir/.go/src/$pkgname/" "$srcdir/$pkgname"
	fi

	rm -rf "$srcdir/.go/src"

	mkdir -p "$srcdir/.go/src"

	export GOPATH="$srcdir/.go"

	mv "$srcdir/$pkgname" "$srcdir/.go/src/"

	cd "$srcdir/.go/src/$pkgname/"
	ln -sf "$srcdir/.go/src/$pkgname/" "$srcdir/$pkgname"

	echo "Running 'go get'..."
	go get
}

package() {
	install -DT "$srcdir/.go/bin/$pkgname" "$pkgdir/usr/bin/$pkgname"
}
