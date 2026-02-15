package designer

const systemPrompt = `You are the Product Designer Agent for MicroFoundry, a micro-version of CloudFoundry that runs on Kubernetes.

Your job is to:
1. Assess whether an Epic requires any web-based user interface
2. If yes, design and generate a self-contained HTML mockup with inline CSS/JS

When assessing UI needs, consider:
- Does this Epic involve user-facing features (dashboards, forms, settings pages)?
- Would a CLI-only experience be sufficient?
- Is there a monitoring or observability UI needed?
- Are there configuration or management UIs?

If UI IS needed, generate a complete, self-contained HTML file that:
- Uses modern CSS (flexbox/grid, CSS variables for theming)
- Includes responsive design
- Uses a clean, professional design language
- Has all CSS inline (no external dependencies)
- Shows realistic mock data
- Is ready to use as a design reference for developers
- Follows MicroFoundry's identity (modern cloud-native tooling)

Output format:
- First line must be either "UI_NEEDED: yes" or "UI_NEEDED: no"
- If yes, follow with the complete HTML design`

const assessmentPrompt = `Review this Epic and determine if a web user interface is needed.

Reply with ONLY "yes" or "no" on the first line, followed by a brief explanation.`

func buildDesignerPrompt(issueBody string, comments []string) string {
	prompt := `# Epic to Assess for UI Requirements

` + issueBody + `

## Agent Comments
`
	for _, c := range comments {
		prompt += c + "\n\n---\n\n"
	}

	prompt += `
---

` + assessmentPrompt

	return prompt
}

func buildMockupPrompt(issueBody string, comments []string) string {
	prompt := `# Epic Requiring UI Design

` + issueBody + `

## Agent Comments
`
	for _, c := range comments {
		prompt += c + "\n\n---\n\n"
	}

	prompt += `
---

Generate a complete, self-contained HTML mockup for the UI components needed by this Epic.
The HTML should be a single file with all CSS inline. Use modern design practices.
Output ONLY the HTML code, starting with <!DOCTYPE html>.`

	return prompt
}
