package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
)

const (
	spaceMargin                  = 10
	errorExitCode                = 1
	okExitCode                   = 0
	expectedDataChunksCount      = 3
	defaultMdElementSize         = 10
	conditionType                = "Ready"
	scriptName                   = "extract_conditions_data.sh"
	expectedMdTableElementsCount = 8
)

type reasonMetadata struct {
	groupOrder      int
	crState         string
	conditionType   string
	conditionStatus bool
	conditionReason string
	remark          string
}

func main() {
	dataForProcessing := extractData()
	dataChunks := strings.Split(dataForProcessing, "====")
	if len(dataChunks) != expectedDataChunksCount {
		fmt.Println(fmt.Sprintf("'%s' data output failed, it should contain 3 elements", scriptName))
		os.Exit(errorExitCode)
	}

	constReasons := getConstReasons(dataChunks[0])

	errors, reasonsMetadata := getAndValidateReasonsMetadata(dataChunks[1])
	if len(errors) > 0 {
		printErrors(errors)
		os.Exit(errorExitCode)
	}

	errors = checkIfConstsAndMetadataAreInSync(constReasons, reasonsMetadata)
	if len(errors) > 0 {
		fmt.Println("The declared reasons in const Go section are out out sync with Reasons metadata")
		printErrors(errors)
		os.Exit(errorExitCode)
	}

	errors, mdTableContent := mdTableToStruct(dataChunks[2])
	if len(errors) > 0 {
		fmt.Println("current table in docs is incorrect:")
		printErrors(errors)
		os.Exit(errorExitCode)
	}

	errors = compareContent(mdTableContent, reasonsMetadata)
	if len(errors) > 0 {
		printErrors(errors)
		fmt.Println("Below can be found auto-generated table which contain new changes:")
		fmt.Println(buildMdTable(reasonsMetadata))
		os.Exit(errorExitCode)
	}

	fmt.Println("docs validation OK. go file is in sync with docs.")
	os.Exit(okExitCode)
}

func extractData() string {
	cmd := exec.Command("/bin/sh", fmt.Sprintf("scripts/autodoc/%s", scriptName))
	var cmdOut, cmdErr bytes.Buffer
	cmd.Stdout = &cmdOut
	cmd.Stderr = &cmdErr
	if err := cmd.Run(); err != nil {
		fmt.Println(cmdErr.String())
		os.Exit(errorExitCode)
	}
	return cmdOut.String()
}

func getConstReasons(input string) []string {
	constReasons := make([]string, 0)
	lines := strings.Split(input, "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) > 0 {
			constReasons = append(constReasons, fields[0])
		}
	}
	return constReasons
}

func getAndValidateReasonsMetadata(input string) ([]string, []reasonMetadata) {
	lines := strings.Split(input, "\n")
	reasonsMetadata := make([]reasonMetadata, 0)
	errors := make([]string, 0)
	for _, line := range lines {
		if line == "" {
			continue
		}
		err, lineStructured := tryConvertGoLineToStruct(line)
		if err != nil {
			errors = append(errors, err.Error())
			continue
		}
		if lineStructured != nil {
			reasonsMetadata = append(reasonsMetadata, *lineStructured)
		}
	}

	return errors, reasonsMetadata
}

func checkIfConstsAndMetadataAreInSync(constReasons []string, reasonsMetadata []reasonMetadata) []string {
	checkIfConstReasonHaveMetadata := func(constReason string) bool {
		for _, reasonMetadata := range reasonsMetadata {
			if reasonMetadata.conditionReason == constReason {
				return true
			}
		}
		return false
	}
	errors := make([]string, 0)

	for _, constReason := range constReasons {
		if !checkIfConstReasonHaveMetadata(constReason) {
			errors = append(errors, fmt.Sprintf("there is a Reason = (%s) declared in const scope, but there is no matching metadata for it", constReason))
		}
	}
	return errors
}

func mdTableToStruct(tableMd string) ([]string, []reasonMetadata) {
	var errors []string
	mdRows := strings.Split(tableMd, "\n")
	mdRows = mdRows[2 : len(mdRows)-1]
	structuredData := make([]reasonMetadata, 0)
	for _, mdRow := range mdRows {
		cleanLine := strings.Split(mdRow, "|")
		numberOfElements := len(cleanLine)
		if numberOfElements != expectedMdTableElementsCount {
			errors = append(errors, fmt.Sprintf("%s have incorrect number of elements, it has %d but it should have %d", cleanLine, numberOfElements, expectedMdTableElementsCount))
			continue
		}

		cleanLine = cleanLine[1 : len(cleanLine)-1]

		crState := cleanLine[1]
		cleanString(&crState)

		conditionType := cleanLine[2]
		cleanString(&conditionType)

		conditionStatusString := cleanLine[3]
		cleanString(&conditionStatusString)
		conditionStatus, err := strconv.ParseBool(conditionStatusString)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s -> cannot parse condition status", cleanLine))
			continue
		}

		conditionReason := cleanLine[4]
		cleanString(&conditionReason)

		remark := cleanLine[5]
		remark = strings.TrimSpace(remark)

		metadata := reasonMetadata{
			groupOrder:      detectGroupOrder(crState),
			crState:         crState,
			conditionType:   conditionType,
			conditionStatus: conditionStatus,
			conditionReason: conditionReason,
			remark:          remark,
		}
		structuredData = append(structuredData, metadata)
	}

	return errors, structuredData
}

func compareContent(currentTableStructured []reasonMetadata, newTableStructured []reasonMetadata) []string {
	checkIfValuesAreSynced := func(new, old, reason string) string {
		if new != old {
			return fmt.Sprintf("Docs are not synced with Go code, difference detected in reason (%s), current value in docs is (%s) but newer in Go code is (%s)", reason, new, old)
		}
		return ""
	}
	errors := make([]string, 0)

	for _, newRow := range newTableStructured {
		foundReasonInDoc := false
		for _, currentRow := range currentTableStructured {
			if newRow.conditionReason == currentRow.conditionReason {
				foundReasonInDoc = true

				if validationMessage := checkIfValuesAreSynced(currentRow.remark, newRow.remark, newRow.conditionReason); validationMessage != "" {
					errors = append(errors, validationMessage)
				}

				if validationMessage := checkIfValuesAreSynced(strconv.FormatBool(currentRow.conditionStatus), strconv.FormatBool(newRow.conditionStatus), newRow.conditionReason); validationMessage != "" {
					errors = append(errors, validationMessage)
				}

				if validationMessage := checkIfValuesAreSynced(currentRow.crState, newRow.crState, newRow.conditionReason); validationMessage != "" {
					errors = append(errors, validationMessage)
				}

				if validationMessage := checkIfValuesAreSynced(currentRow.conditionType, newRow.conditionType, newRow.conditionReason); validationMessage != "" {
					errors = append(errors, validationMessage)
				}

				break
			}
		}

		if !foundReasonInDoc {
			errors = append(errors, fmt.Sprintf("Reason (%s) not found in docs.", newRow.conditionReason))
		}
	}
	return errors
}

func buildMdTable(reasonsMetadata []reasonMetadata) string {
	renderMdElement := func(length int, content string, spaceFiller string) string {
		length = length - len(content)
		var element strings.Builder
		element.WriteString(content)
		for i := 0; i < length+spaceMargin; i++ {
			element.WriteString(spaceFiller)
		}
		return element.String()
	}

	sort.Slice(reasonsMetadata, func(i, j int) bool {
		if reasonsMetadata[i].groupOrder != reasonsMetadata[j].groupOrder {
			return reasonsMetadata[i].groupOrder < reasonsMetadata[j].groupOrder
		}
		return reasonsMetadata[i].conditionReason < reasonsMetadata[j].conditionReason
	})

	longestConditionReasons := 0
	longestRemark := 0
	for _, reasonMetadata := range reasonsMetadata {
		tempLongestConditionReasons := len(reasonMetadata.conditionReason)
		if tempLongestConditionReasons > longestConditionReasons {
			longestConditionReasons = tempLongestConditionReasons
		}
		tempLongestRemark := len(reasonMetadata.remark)
		if tempLongestRemark > longestRemark {
			longestRemark = tempLongestRemark
		}
	}

	var mdTable strings.Builder

	mdTable.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s | %s |\n",
		renderMdElement(defaultMdElementSize, "No.", " "),
		renderMdElement(defaultMdElementSize, "CR state", " "),
		renderMdElement(defaultMdElementSize, "Condition type", " "),
		renderMdElement(defaultMdElementSize, "Condition status", " "),
		renderMdElement(longestConditionReasons, "Condition reason", " "),
		renderMdElement(longestRemark, "Remark", " ")))

	mdTable.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s | %s |\n",
		renderMdElement(defaultMdElementSize, "", "-"),
		renderMdElement(defaultMdElementSize, "", "-"),
		renderMdElement(defaultMdElementSize, "", "-"),
		renderMdElement(defaultMdElementSize, "", "-"),
		renderMdElement(longestConditionReasons, "", "-"),
		renderMdElement(longestRemark, "", "-")))

	lineNumber := 1
	for _, row := range reasonsMetadata {
		mdTable.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s | %s |\n",
			renderMdElement(defaultMdElementSize, strconv.Itoa(lineNumber), " "),
			renderMdElement(defaultMdElementSize, row.crState, " "),
			renderMdElement(defaultMdElementSize, row.conditionType, " "),
			renderMdElement(defaultMdElementSize, strconv.FormatBool(row.conditionStatus), " "),
			renderMdElement(longestConditionReasons, row.conditionReason, " "),
			renderMdElement(longestRemark, row.remark, " ")))
		lineNumber++
	}

	return mdTable.String()
}

func tryConvertGoLineToStruct(goLine string) (error, *reasonMetadata) {
	if goLine == "" {
		return fmt.Errorf("empty goLine given"), nil
	}
	goLine = strings.TrimSpace(goLine)
	parts := strings.Split(goLine, "//")
	if len(parts) != 2 {
		return fmt.Errorf("in goLine (%s) there is no comment section (//) included, comment section should have following format (//CRState;Remark)", goLine), nil
	}

	words := strings.Fields(parts[0])
	if len(words) != 5 {
		return fmt.Errorf("goLine (%s) is badly structured, it should have following format (Reason: Metadata, //CRState;Remark", goLine), nil
	}

	comments := strings.Split(parts[1], ";")
	if len(comments) != 2 {
		return fmt.Errorf("comment in goLine (%s) is badly structured, it should have following format (//CRState;Remark)", goLine), nil
	}

	reason := words[0]
	cleanString(&reason)

	state := comments[0]
	cleanString(&state)

	remark := comments[1]
	remark = strings.TrimSpace(remark)

	return nil, &reasonMetadata{
		groupOrder:      detectGroupOrder(state),
		crState:         state,
		conditionType:   conditionType,
		conditionStatus: getConditionStatus(state, conditionType),
		conditionReason: reason,
		remark:          remark,
	}
}

func getConditionStatus(state, conditionType string) bool {
	return state == "Ready" && conditionType == "Ready"
}

func detectGroupOrder(state string) int {
	switch state {
	case "Ready":
		return 1
	case "Processing":
		return 2
	case "Deleting":
		return 3
	case "Error":
		return 4
	case "Warning":
		return 5
	case "NA":
		return 6
	default:
		return 7
	}
}

func printErrors(errors []string) {
	for _, error := range errors {
		fmt.Println(fmt.Sprintf("validation failed! -> %s", error))
	}
}

func cleanString(s *string) {
	*s = strings.Replace(*s, " ", "", -1)
	*s = strings.Replace(*s, ":", "", -1)
	*s = strings.Replace(*s, "/", "", -1)
	*s = strings.Replace(*s, ",", "", -1)
}
