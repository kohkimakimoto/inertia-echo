package inertia

import "encoding/json"

// Vite integration for builtin renderer
// see https://vitejs.dev/guide/backend-integration.html

type ViteManifest map[string]interface{}

func ViteEntry(manifest ViteManifest) func(key string) map[string]interface{} {
	return func(key string) map[string]interface{} {
		if entry, ok := manifest[key]; ok {
			if entry, ok := entry.(map[string]interface{}); ok {
				return entry
			}
		}
		return nil
	}
}

func ParseViteManifest(data []byte) (ViteManifest, error) {
	var manifest ViteManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	return manifest, nil
}
