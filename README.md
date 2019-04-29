# tinyletter to Hugo 
Tool for the migration of articles from a tinyletter.com account to a static website generator with markdown like Hugo.

Here is a preview from my own migration (from tinyletter to Hugo):
![example](./preview.png)

## Features:
* transform HTML posts to markdown
* even if one article fails it keeps going
* stories are ordered by date (`year-month-day_slug`)
* custom `.Params`: image, images, slug, subtitle
* minimal HTML cleanup (removed empty URLs, duplicate title and so on)

## Usage 

1. Download your tinyletter data
2. Unzip it
3. Download our binary (see Releases)
4. Run something like `./tinylettertohugo /path/to/export/posts /path/to/hugo/content/ posts`


### Build / Contribute
You need Bash, Go 1.11+

See Issues for a list of TODOs.
