package feeds

import "encoding/xml"

const xmlNamespace = "http://www.w3.org/XML/1998/namespace"
const rssContentNamespace = "http://purl.org/rss/1.0/modules/content/"

func hasXMLAttribute(start xml.StartElement, local string) bool {
	for _, attr := range start.Attr {
		if attr.Name.Space == xmlNamespace && attr.Name.Local == local {
			return true
		}
	}
	return false
}
