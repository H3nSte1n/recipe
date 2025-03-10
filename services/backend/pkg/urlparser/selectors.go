// pkg/urlparser/content_filters.go
package urlparser

// UnwantedSelectors defines HTML elements and classes that should be removed
var UnwantedSelectors = []string{
	// Basic HTML elements
	"script", // JavaScript
	"style",  // CSS
	"link",   // External resources
	"meta",   // Meta tags
	"iframe", // Embedded content
	"header", // Site header
	"footer", // Site footer
	"nav",    // Navigation
	"aside",  // Sidebars

	// Common class-based elements
	".sidebar",       // Sidebars
	".comments",      // Comments section
	".advertisement", // Ads
	".social-share",  // Social sharing
	".related-posts", // Related content
	".popup",         // Popups
	".modal",         // Modals
	".newsletter",    // Newsletter signup
	".breadcrumb",    // Breadcrumbs
	".pagination",    // Pagination
	".widget",        // Widgets
	".banner",        // Banners

	// Wildcard selectors
	"[class*='menu']",    // Menus
	"[class*='nav']",     // Navigation
	"[class*='share']",   // Share buttons
	"[class*='print']",   // Print buttons
	"[class*='save']",    // Save buttons
	"[class*='rating']",  // Rating widgets
	"[class*='comment']", // Comments
	"[class*='author']",  // Author bios
	"[class*='sidebar']", // Sidebars
	"[class*='widget']",  // Widgets
	"[class*='ad-']",     // Ads
	"[id*='ad-']",        // Ads

	// Accessibility and media
	"[aria-hidden='true']", // Hidden elements
	"img",                  // Images (handle separately if needed)

	// Additional common unwanted elements
	".cookie-notice",         // Cookie notices
	".notification",          // Notifications
	".alert",                 // Alerts
	".search",                // Search boxes
	".toolbar",               // Toolbars
	".skiplink",              // Skip links
	".skip-link",             // Skip links
	"[role='banner']",        // Header roles
	"[role='navigation']",    // Navigation roles
	"[role='complementary']", // Sidebar roles
}

// MainContentSelectors defines selectors for finding the main content
var MainContentSelectors = []string{
	"main",
	"article",
	".content",
	".post-content",
	".entry-content",
	"[class*='content']",
	"[class*='article']",
	".post-body",
	".entry",
	"#main-content",
	".main-content",
}
