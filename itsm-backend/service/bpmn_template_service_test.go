package service

import (
	"bytes"
	"encoding/xml"
	"io"
	"strings"
	"testing"
	"unicode"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListTemplates_DevApprovalOpsFlow(t *testing.T) {
	svc := NewBPMNTemplateService(nil)
	templates, err := svc.listTemplates()
	require.NoError(t, err)

	var found bool
	for _, tpl := range templates {
		if tpl.ID == "dev_approval_ops_flow" {
			found = true
			assert.Equal(t, "Development Approval and Operations Workflow", tpl.Name)
			assert.Equal(t, "ticket", tpl.Category)
			assert.Equal(t, "approval", tpl.SubCategory)
			assert.Equal(t, "Three-stage workflow: development submission, supervisor approval, and operations execution", tpl.Description)
			break
		}
	}
	assert.True(t, found, "dev_approval_ops_flow template should be listed")
}

func TestEmbeddedBPMNTemplatesReadableOnEveryPlatform(t *testing.T) {
	svc := NewBPMNTemplateService(nil)
	templates, err := svc.GetTemplateList()
	if err != nil {
		t.Fatal(err)
	}
	if len(templates) == 0 {
		t.Fatal("expected built-in workflow templates")
	}
	for _, template := range templates {
		t.Run(template.ID, func(t *testing.T) {
			content, err := svc.GetTemplateContent(template.ID)
			if err != nil {
				t.Fatalf("read embedded template: %v", err)
			}
			if !strings.Contains(string(content), "definitions") {
				t.Fatal("expected BPMN definitions in template content")
			}
			containsChinese := func(value string) bool {
				return strings.ContainsFunc(value, func(r rune) bool { return unicode.Is(unicode.Han, r) })
			}
			require.False(t, containsChinese(template.Name), "template name must be English")
			require.NotEqual(t, template.ID, template.Name, "built-in variants must use their display title")
			require.False(t, containsChinese(template.Description), "template description must be English")
			decoder := xml.NewDecoder(bytes.NewReader(content))
			var displayText bool
			for {
				token, err := decoder.Token()
				if err == io.EOF {
					break
				}
				require.NoError(t, err)
				switch element := token.(type) {
				case xml.StartElement:
					displayText = element.Name.Local == "documentation" || element.Name.Local == "text"
					for _, attr := range element.Attr {
						if attr.Name.Local == "name" {
							require.False(t, containsChinese(attr.Value), "diagram label %q must be English", attr.Value)
							if element.Name.Local == "metaData" && attr.Value == "description" {
								displayText = true
							}
						}
					}
				case xml.CharData:
					if displayText {
						require.False(t, containsChinese(string(element)), "diagram documentation must be English")
					}
				case xml.EndElement:
					displayText = false
				}
			}
		})
	}
}
