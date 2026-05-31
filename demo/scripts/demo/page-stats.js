// Summary statistics about the current page DOM.
return {
    url:        location.href,
    title:      document.title,
    nodes:      document.querySelectorAll('*').length,
    links:      document.querySelectorAll('a').length,
    images:     document.querySelectorAll('img').length,
    scripts:    document.querySelectorAll('script').length,
    headings:   {
        h1: document.querySelectorAll('h1').length,
        h2: document.querySelectorAll('h2').length,
        h3: document.querySelectorAll('h3').length
    },
    viewport:   { w: window.innerWidth, h: window.innerHeight },
    docHeight:  document.documentElement.scrollHeight
};
