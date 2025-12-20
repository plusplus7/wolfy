cd ..
rm -rf static/css
rm -rf static/js
rm -rf static/index.html

cd ../wolfy_web
npm run build
cp -r dist/static/css ../wolfy/static/css
cp -r dist/static/js ../wolfy/static/js
cp -r dist/index.html ../wolfy/static/index.html
