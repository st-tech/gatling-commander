/*
Copyright &copy; ZOZO, Inc.

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the “Software”), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included
in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

// Package spreadsheeet implements operator to write loadtest result to spreadsheets.
package spreadsheet

import (
	"context"
	"fmt"
	"reflect"

	"google.golang.org/api/sheets/v4"
)

/*
SheetNotFoundError implements Error and Is methods.

This error is returned when got sheets is nothing.
*/
type SheetNotFoundError struct {
	sheetName string
}

/*
spreadsheetOperator hold updated spreadsheet.

Operating spreadsheet by this operators method, hold latest spreadsheet.
Call spreadsheet api need to prepare BatchUpdateSpreadsheetRequest
and given it as doBatchUpdate method argument.
And run doBatchUpdate method.
*/
type spreadsheetOperator struct {
	ctx         context.Context
	service     *sheets.Service
	spreadsheet *sheets.Spreadsheet
}

// loadtestCommonSettingRow hold row value which is common setting of loadtest.
type loadtestCommonSettingRow struct {
	imageURL      string
	serviceName   string
	targetLatency string
}

/*
gatlingResultRow hold row value which has gatling report value and calcurated container metrics.

The report example table is below.

| subName |                  condition                  | duration (s) | concurrency (req/s) |

|  4rps   | ENV=dev, ... ,FEATURE_CACHE_HIT_RATIO=0.38, |      180     |         40          |

| max (ms) | mean (ms) | 50%ile latency (ms) | 75%ile latency (ms) | 95%ile latency (ms) |

|   1107   |     51    |          31         |          55         |          64         |

| 99%ile latency (ms) | failed | t < 800 | 800 < t <= 1200 | 1200 < t | cpu usage mean (%) | memory usage mean (%) |

|          674        |    0   |   100   |         0       |     0    |         15.5       |         22.3          |
*/
type loadtestReportRow struct {
	subName                                               string
	condition                                             string
	duration                                              string
	concurrency                                           string
	maxLatency                                            float64
	meanLatency                                           float64
	fiftiethPercentilesLatency                            float64
	seventyFifthPercentilesLatency                        float64
	nintyFifthPercentilesLatency                          float64
	nintyNinthPercentilesLatency                          float64
	failedPercentage                                      float64
	underEightHundredMilliSecPercentage                   float64
	eightHundredToOneThousandTwoHundredMilliSecPercentage float64
	overOneThousandTwoHundredMilliSecPercentage           float64
	cpuUsePercentage                                      float64
	memoryUsePercentage                                   float64
}

// NewLoadtestCommonSettingRow creates loadtestCommonSettingRow objects.
func NewLoadtestCommonSettingRow(imageURL, serviceName, targetLatency string) loadtestCommonSettingRow {
	return loadtestCommonSettingRow{
		imageURL:      imageURL,
		serviceName:   serviceName,
		targetLatency: targetLatency,
	}
}

// NewLoadtestReportRow creates loadtestReportRow objects.
func NewLoadtestReportRow(
	subName, condition, duration, concurrency string,
	maxLatency, meanLatency, fiftiethPercentilesLatency, seventyFifthPercentilesLatency,
	nintyFifthPercentilesLatency, nintyNinthPercentilesLatency, failedPercentage,
	underEightHundredMilliSecPercentage, eightHundredToOneThousandTwoHundredMilliSecPercentage,
	overOneThousandTwoHundredMilliSecPercentage, cpuUsePercentage, memoryUsePercentage float64,
) loadtestReportRow {
	return loadtestReportRow{
		subName:                             subName,
		condition:                           condition,
		concurrency:                         concurrency,
		duration:                            duration,
		maxLatency:                          maxLatency,
		meanLatency:                         meanLatency,
		fiftiethPercentilesLatency:          fiftiethPercentilesLatency,
		seventyFifthPercentilesLatency:      seventyFifthPercentilesLatency,
		nintyFifthPercentilesLatency:        nintyFifthPercentilesLatency,
		nintyNinthPercentilesLatency:        nintyNinthPercentilesLatency,
		failedPercentage:                    failedPercentage,
		underEightHundredMilliSecPercentage: underEightHundredMilliSecPercentage,
		eightHundredToOneThousandTwoHundredMilliSecPercentage: eightHundredToOneThousandTwoHundredMilliSecPercentage,
		overOneThousandTwoHundredMilliSecPercentage:           overOneThousandTwoHundredMilliSecPercentage,
		cpuUsePercentage:    cpuUsePercentage,
		memoryUsePercentage: memoryUsePercentage,
	}
}

// NewSpreadsheetOperator returns initialized spreadsheetOperator.
func NewSpreadsheetOperator(ctx context.Context, spreadsheetId string) (*spreadsheetOperator, error) {
	var op spreadsheetOperator
	srv, err := sheets.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create new sheets service, %v", err)
	}
	op.ctx = ctx
	op.service = srv
	targetSpreadsheet, err := op.service.Spreadsheets.Get(spreadsheetId).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get target spreadsheet, %v", err)
	}
	op.spreadsheet = targetSpreadsheet
	return &op, nil
}

// AddSheet method add new sheet to spreadsheet and returns added sheet.
func (op *spreadsheetOperator) AddSheet(sheetTitle string) (*sheets.Sheet, error) {
	createNewSheetReq := &sheets.BatchUpdateSpreadsheetRequest{
		IncludeSpreadsheetInResponse: true,
		Requests: []*sheets.Request{
			&sheets.Request{
				AddSheet: &sheets.AddSheetRequest{
					Properties: &sheets.SheetProperties{
						Title: sheetTitle,
					},
				},
			},
		},
	}
	err := op.doBatchUpdate(createNewSheetReq)
	if err != nil {
		return &sheets.Sheet{}, err
	}
	foundSheet, err := op.FindSheet(sheetTitle)
	if err != nil {
		return &sheets.Sheet{}, err
	}
	return foundSheet, nil
}

// FindSheet method find target sheet and returns found sheet.
func (op *spreadsheetOperator) FindSheet(sheetTitle string) (*sheets.Sheet, error) {
	sheetsMap := make(map[string]*sheets.Sheet)
	// Google Spreadsheet always have more than 1 sheet.
	for _, sheet := range op.spreadsheet.Sheets {
		sheetsMap[sheet.Properties.Title] = sheet
	}

	foundSheet, exist := sheetsMap[sheetTitle]
	if !exist {
		return &sheets.Sheet{}, &SheetNotFoundError{sheetName: sheetTitle}
	}
	return foundSheet, nil
}

// SetCellName method set column header to sheet and returns updated sheet.
func (op *spreadsheetOperator) SetColumnHeader(targetSheet *sheets.Sheet) (*sheets.Sheet, error) {
	// sheets.ExtendedValue StringValue field need pointer,
	// so assign string value to var and give its pointer to map value.
	commonSettingHeader := struct {
		imageURLColumnName      string
		serviceNameColumnName   string
		targetLatencyColumnName string
	}{
		imageURLColumnName:      "imageURL",
		serviceNameColumnName:   "serviceName",
		targetLatencyColumnName: "targetLatency",
	}
	commonSettingHeaderColumnNum := reflect.TypeOf(commonSettingHeader).NumField()

	gatlingReportHeader := struct {
		subNameColumnName                                                     string
		conditionColumnName                                                   string
		durationColumnName                                                    string
		concurrencyColumnName                                                 string
		maxLatencyColumnName                                                  string
		meanLatencyColumnName                                                 string
		fiftiethPercentileColumnName                                          string
		seventyfifthPercentileColumnName                                      string
		ninetyfifthPercentileColumnName                                       string
		ninetyninthPercentileColumnName                                       string
		failedPercentageColumnName                                            string
		underEightHundredMilliSecCountColumnName                              string
		betweenFromEightHundredToOneThousandTwoHundredMilliSecCountColumnName string
		overOneThousandTwoHundredMilliSecCountColumnName                      string
		cpuUsePercentageColumnName                                            string
		memoryUsePercentageColumnName                                         string
	}{
		subNameColumnName:                        "subName",
		conditionColumnName:                      "condition",
		durationColumnName:                       "duration (s)",
		concurrencyColumnName:                    "concurrency (req/s)",
		maxLatencyColumnName:                     "max (ms)",
		meanLatencyColumnName:                    "mean (ms)",
		fiftiethPercentileColumnName:             "50%ile latency (ms)",
		seventyfifthPercentileColumnName:         "75%ile latency (ms)",
		ninetyfifthPercentileColumnName:          "95%ile latency (ms)",
		ninetyninthPercentileColumnName:          "99%ile latency (ms)",
		failedPercentageColumnName:               "failed",
		underEightHundredMilliSecCountColumnName: "t < 800",
		betweenFromEightHundredToOneThousandTwoHundredMilliSecCountColumnName: "800 < t <= 1200",
		overOneThousandTwoHundredMilliSecCountColumnName:                      "1200 < t",
		cpuUsePercentageColumnName:                                            "cpu usage mean (%)",
		memoryUsePercentageColumnName:                                         "memory usage mean (%)",
	}
	gatlingReportHeaderColumnNum := reflect.TypeOf(gatlingReportHeader).NumField()

	setCellNameReq := &sheets.BatchUpdateSpreadsheetRequest{
		IncludeSpreadsheetInResponse: true,
		Requests: []*sheets.Request{
			&sheets.Request{
				UpdateCells: &sheets.UpdateCellsRequest{
					Fields: "userEnteredValue",
					Range: &sheets.GridRange{
						SheetId:          targetSheet.Properties.SheetId,
						StartRowIndex:    0,
						EndRowIndex:      1,
						StartColumnIndex: 0,
						// (caution) Index count from 0 and `EndIndex` should be specified
						// as the index value of the last column plus 1.
						EndColumnIndex: int64(
							commonSettingHeaderColumnNum,
						),
					},
					Rows: []*sheets.RowData{
						&sheets.RowData{
							Values: []*sheets.CellData{
								{
									UserEnteredValue: &sheets.ExtendedValue{
										StringValue: &commonSettingHeader.imageURLColumnName,
									},
								},
								{
									UserEnteredValue: &sheets.ExtendedValue{
										StringValue: &commonSettingHeader.serviceNameColumnName,
									},
								},
								{
									UserEnteredValue: &sheets.ExtendedValue{
										StringValue: &commonSettingHeader.targetLatencyColumnName,
									},
								},
							},
						},
					},
				},
			},
			&sheets.Request{
				UpdateCells: &sheets.UpdateCellsRequest{
					Fields: "userEnteredValue",
					Range: &sheets.GridRange{
						SheetId:          targetSheet.Properties.SheetId,
						StartRowIndex:    3,
						EndRowIndex:      5,
						StartColumnIndex: 0,
						EndColumnIndex: int64(
							// (caution) Index count from 0 and `EndIndex` should be specified
							// as the index value of the last column plus 1.
							gatlingReportHeaderColumnNum,
						),
					},
					Rows: []*sheets.RowData{
						&sheets.RowData{
							Values: []*sheets.CellData{
								{
									UserEnteredValue: &sheets.ExtendedValue{
										StringValue: &gatlingReportHeader.subNameColumnName,
									},
								},
								{
									UserEnteredValue: &sheets.ExtendedValue{
										StringValue: &gatlingReportHeader.conditionColumnName,
									},
								},
								{
									UserEnteredValue: &sheets.ExtendedValue{
										StringValue: &gatlingReportHeader.durationColumnName,
									},
								},
								{
									UserEnteredValue: &sheets.ExtendedValue{
										StringValue: &gatlingReportHeader.concurrencyColumnName,
									},
								},
								{
									UserEnteredValue: &sheets.ExtendedValue{
										StringValue: &gatlingReportHeader.maxLatencyColumnName,
									},
								},
								{
									UserEnteredValue: &sheets.ExtendedValue{
										StringValue: &gatlingReportHeader.meanLatencyColumnName,
									},
								},
								{
									UserEnteredValue: &sheets.ExtendedValue{
										StringValue: &gatlingReportHeader.fiftiethPercentileColumnName,
									},
								},
								{
									UserEnteredValue: &sheets.ExtendedValue{
										StringValue: &gatlingReportHeader.seventyfifthPercentileColumnName,
									},
								},
								{
									UserEnteredValue: &sheets.ExtendedValue{
										StringValue: &gatlingReportHeader.ninetyfifthPercentileColumnName,
									},
								},
								{
									UserEnteredValue: &sheets.ExtendedValue{
										StringValue: &gatlingReportHeader.ninetyninthPercentileColumnName,
									},
								},
								{
									UserEnteredValue: &sheets.ExtendedValue{
										StringValue: &gatlingReportHeader.failedPercentageColumnName,
									},
								},
								{
									UserEnteredValue: &sheets.ExtendedValue{
										StringValue: &gatlingReportHeader.underEightHundredMilliSecCountColumnName,
									},
								},
								{
									UserEnteredValue: &sheets.ExtendedValue{
										StringValue: &gatlingReportHeader.betweenFromEightHundredToOneThousandTwoHundredMilliSecCountColumnName,
									},
								},
								{
									UserEnteredValue: &sheets.ExtendedValue{
										StringValue: &gatlingReportHeader.overOneThousandTwoHundredMilliSecCountColumnName,
									},
								},
								{
									UserEnteredValue: &sheets.ExtendedValue{
										StringValue: &gatlingReportHeader.cpuUsePercentageColumnName,
									},
								},
								{
									UserEnteredValue: &sheets.ExtendedValue{
										StringValue: &gatlingReportHeader.memoryUsePercentageColumnName,
									},
								},
							},
						},
					},
				},
			},
		},
	}
	err := op.doBatchUpdate(setCellNameReq)
	if err != nil {
		return &sheets.Sheet{}, err
	}
	foundSheet, err := op.FindSheet(targetSheet.Properties.Title)
	if err != nil {
		return &sheets.Sheet{}, fmt.Errorf("set cell name but sheet not found")
	}
	return foundSheet, nil
}

// SetLoadtestCommonSettingValue method set common setting value to column.
func (op *spreadsheetOperator) SetLoadtestCommonSettingValue(
	row loadtestCommonSettingRow,
	targetSheet *sheets.Sheet,
) (*sheets.Sheet, error) {
	loadtestCommonSettingColumnNum := reflect.TypeOf(row).NumField()
	setLoadtestCommonSettingCellValueReq := &sheets.BatchUpdateSpreadsheetRequest{
		IncludeSpreadsheetInResponse: true,
		Requests: []*sheets.Request{
			&sheets.Request{
				UpdateCells: &sheets.UpdateCellsRequest{
					Fields: "userEnteredValue",
					Range: &sheets.GridRange{
						SheetId:          targetSheet.Properties.SheetId,
						StartRowIndex:    1,
						EndRowIndex:      3,
						StartColumnIndex: 0,
						EndColumnIndex:   int64(loadtestCommonSettingColumnNum),
					},
					Rows: []*sheets.RowData{
						&sheets.RowData{
							Values: []*sheets.CellData{
								{
									UserEnteredValue: &sheets.ExtendedValue{
										StringValue: &row.imageURL,
									},
								},
								{
									UserEnteredValue: &sheets.ExtendedValue{
										StringValue: &row.serviceName,
									},
								},
								{
									UserEnteredValue: &sheets.ExtendedValue{
										StringValue: &row.targetLatency,
									},
								},
							},
						},
					},
				},
			},
		},
	}
	err := op.doBatchUpdate(setLoadtestCommonSettingCellValueReq)
	if err != nil {
		return &sheets.Sheet{}, err
	}
	foundSheet, err := op.FindSheet(targetSheet.Properties.Title)
	if err != nil {
		return &sheets.Sheet{}, fmt.Errorf("loadtest common setting was added but sheet not found")
	}
	return foundSheet, nil
}

// AppendLoadtestReportRow append report of each loadtest to the end of row in target sheet.
func (op *spreadsheetOperator) AppendLoadtestReportRow(
	row loadtestReportRow,
	targetSheet *sheets.Sheet,
) (*sheets.Sheet, error) {
	loadtestReportColumnNum := reflect.TypeOf(row).NumField()
	readRange := fmt.Sprintf("%v!A:M", targetSheet.Properties.Title)
	existingRowCount, err := op.getRowCount(readRange)
	if err != nil {
		return &sheets.Sheet{}, err
	}
	addLoadtestReportRowReq := &sheets.BatchUpdateSpreadsheetRequest{
		IncludeSpreadsheetInResponse: true,
		Requests: []*sheets.Request{
			&sheets.Request{
				UpdateCells: &sheets.UpdateCellsRequest{
					Fields: "userEnteredValue",
					Range: &sheets.GridRange{
						SheetId:          targetSheet.Properties.SheetId,
						StartRowIndex:    existingRowCount,
						EndRowIndex:      existingRowCount + 2,
						StartColumnIndex: 0,
						EndColumnIndex:   int64(loadtestReportColumnNum),
					},
					Rows: []*sheets.RowData{
						&sheets.RowData{
							Values: []*sheets.CellData{
								{
									UserEnteredValue: &sheets.ExtendedValue{
										StringValue: &row.subName,
									},
								},
								{
									UserEnteredValue: &sheets.ExtendedValue{
										StringValue: &row.condition,
									},
								},
								{
									UserEnteredValue: &sheets.ExtendedValue{
										StringValue: &row.duration,
									},
								},
								{
									UserEnteredValue: &sheets.ExtendedValue{
										StringValue: &row.concurrency,
									},
								},
								{
									UserEnteredValue: &sheets.ExtendedValue{
										NumberValue: &row.maxLatency,
									},
								},
								{
									UserEnteredValue: &sheets.ExtendedValue{
										NumberValue: &row.meanLatency,
									},
								},
								{
									UserEnteredValue: &sheets.ExtendedValue{
										NumberValue: &row.fiftiethPercentilesLatency,
									},
								},
								{
									UserEnteredValue: &sheets.ExtendedValue{
										NumberValue: &row.seventyFifthPercentilesLatency,
									},
								},
								{
									UserEnteredValue: &sheets.ExtendedValue{
										NumberValue: &row.nintyFifthPercentilesLatency,
									},
								},
								{
									UserEnteredValue: &sheets.ExtendedValue{
										NumberValue: &row.nintyNinthPercentilesLatency,
									},
								},
								{
									UserEnteredValue: &sheets.ExtendedValue{
										NumberValue: &row.failedPercentage,
									},
								},
								{
									UserEnteredValue: &sheets.ExtendedValue{
										NumberValue: &row.underEightHundredMilliSecPercentage,
									},
								},
								{
									UserEnteredValue: &sheets.ExtendedValue{
										NumberValue: &row.eightHundredToOneThousandTwoHundredMilliSecPercentage,
									},
								},
								{
									UserEnteredValue: &sheets.ExtendedValue{
										NumberValue: &row.overOneThousandTwoHundredMilliSecPercentage,
									},
								},
								{
									UserEnteredValue: &sheets.ExtendedValue{
										NumberValue: &row.cpuUsePercentage,
									},
								},
								{
									UserEnteredValue: &sheets.ExtendedValue{
										NumberValue: &row.memoryUsePercentage,
									},
								},
							},
						},
					},
				},
			},
		},
	}
	err = op.doBatchUpdate(addLoadtestReportRowReq)
	if err != nil {
		return &sheets.Sheet{}, err
	}
	foundSheet, err := op.FindSheet(targetSheet.Properties.Title)
	if err != nil {
		return &sheets.Sheet{}, fmt.Errorf("loadtest report was added but sheet not found")
	}
	return foundSheet, nil
}

// doBatchUpdate update spreadsheets by given requests. And update spreadsheetOperator's spreadsheet field value by
// updated spreadsheet.
func (op *spreadsheetOperator) doBatchUpdate(req *sheets.BatchUpdateSpreadsheetRequest) error {
	if !req.IncludeSpreadsheetInResponse {
		return fmt.Errorf(
			"invalid input IncludeSpreadsheetInResponse is false",
		)
		// doBatchUpdate always update SpreadsheetOperator field so need to include updated spreadsheet in response.
		// nolint:lll // ref: https://github.com/googleapis/google-api-go-client/blob/113082d14d54f188d1b6c34c652e416592fc51b5/sheets/v4/sheets-gen.go#L1921
	}
	res, err := op.service.Spreadsheets.BatchUpdate(op.spreadsheet.SpreadsheetId, req).Do()
	if err != nil {
		return err
	}
	op.spreadsheet = res.UpdatedSpreadsheet
	return nil
}

/*
getRowCount returns specified targetRange row count.

Specify targetRange in the form of "sheet title!column start:column end". example: "YOUR_SHEET_NAME!A:M"
*/
func (op *spreadsheetOperator) getRowCount(targetRange string) (int64, error) {
	res, err := op.service.Spreadsheets.Values.Get(op.spreadsheet.SpreadsheetId, targetRange).Do()
	if err != nil {
		return 0, err
	}
	return int64(len(res.Values)), nil
}

// Error implements error interface.
func (e *SheetNotFoundError) Error() string {
	return fmt.Sprintf("%s: sheet not found", e.sheetName)
}

// Is method is needed to compare by errors.Is method.
func (e *SheetNotFoundError) Is(target error) bool {
	_, ok := target.(*SheetNotFoundError)
	return ok
}
