package tui

func (ui *Ui) SetHeaderField(fieldName string, value string) {
	hf := &ui.headerFields
	hfv := ui.headerFieldValues

	if _, exists := hfv[fieldName]; !exists {
		*hf = append(*hf, fieldName)
	}
	hfv[fieldName] = value
}
