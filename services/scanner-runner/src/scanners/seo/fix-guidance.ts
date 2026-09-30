/** Concrete examples are templates: use the page's actual content and canonical URL. */
export const SEO_FIXES: Record<string, string> = {
	'missing-title':
		'Add <title>Page topic | Site name</title> inside <head>, with a unique title describing this page.',
	'title-length':
		'Edit <title>Page topic | Site name</title> to clearly describe the page. Aim for roughly 50–60 characters when that preserves meaning.',
	'missing-description':
		'Add <meta name="description" content="A concise, accurate summary of this page."> inside <head>.',
	'description-length':
		'Edit <meta name="description" content="A useful summary of this page and why to visit it.">. Aim for roughly 120–160 characters when appropriate.',
	'missing-h1': 'Add <h1>Main topic of this page</h1> as the visible primary heading.',
	'multiple-h1':
		'Keep the primary page heading as <h1>; use <h2> for sections and <h3> for their subsections.',
	'heading-hierarchy':
		'Nest sections in order: <h1>Page</h1>, <h2>Section</h2>, <h3>Subsection</h3>. Choose heading levels for structure rather than size.',
	'images-missing-alt':
		'Add <img src="…" alt="Describe the information in this image">. Use alt="" for decorative images.',
	'missing-og-tags':
		'Add <meta property="og:title" content="Page title">, <meta property="og:description" content="Page summary"> and <meta property="og:image" content="https://your-site.example/share.jpg"> inside <head>.',
	'missing-canonical':
		'Add <link rel="canonical" href="https://your-site.example/this-page"> inside <head>, using the preferred public URL of this page.',
	'missing-viewport':
		'Add <meta name="viewport" content="width=device-width, initial-scale=1"> inside <head>. Do not disable zoom.',
	'missing-language':
		'Set <html lang="en"> to the language actually used by this page; use the correct language code.',
	'missing-structured-data':
		'When the page represents a supported entity, add <script type="application/ld+json"> with valid schema.org data describing its actual content. Validate it before publishing.',
	'thin-content':
		'Add the information visitors need to complete this page’s purpose. Remove empty placeholder sections; a useful contact or utility page does not need filler text.'
};
