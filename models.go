package goarpa

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// GetQueryParams converts the struct to map[string]string
// The fields tags must have `json:"<name>,string,omitempty"` format for all types, except strings
// The string fields must have: `json:"<name>,omitempty"`. The `json:"<name>,string,omitempty"` tag for string field
// will add additional double quotes.
// "string" tag allows to convert the non-string fields of a structure to map[string]string.
// "omitempty" allows to skip the fields with default values.
func GetQueryParams(s interface{}) (map[string]string, error) {
	// if obj, ok := s.(GetGroupsParams); ok {
	// 	obj.OnMarshal()
	// 	s = obj
	// }
	b, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	var res map[string]string
	err = json.Unmarshal(b, &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// StringOrArray represents a value that can either be a string or an array of strings
type StringOrArray []string

// UnmarshalJSON unmarshals a string or an array object from a JSON array or a JSON string
func (s *StringOrArray) UnmarshalJSON(data []byte) error {
	if len(data) > 1 && data[0] == '[' {
		var obj []string
		if err := json.Unmarshal(data, &obj); err != nil {
			return err
		}
		*s = StringOrArray(obj)
		return nil
	}

	var obj string
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	*s = StringOrArray([]string{obj})
	return nil
}

// MarshalJSON converts the array of strings to a JSON array or JSON string if there is only one item in the array
func (s *StringOrArray) MarshalJSON() ([]byte, error) {
	if len(*s) == 1 {
		return json.Marshal([]string(*s)[0])
	}
	return json.Marshal([]string(*s))
}

// EnforcedString can be used when the expected value is string but Keycloak in some cases gives you mixed types
type EnforcedString string

// UnmarshalJSON modify data as string before json unmarshal
func (s *EnforcedString) UnmarshalJSON(data []byte) error {
	if data[0] != '"' {
		// Escape unescaped quotes
		data = bytes.ReplaceAll(data, []byte(`"`), []byte(`\"`))
		data = bytes.ReplaceAll(data, []byte(`\\"`), []byte(`\"`))

		// Wrap data in quotes
		data = append([]byte(`"`), data...)
		data = append(data, []byte(`"`)...)
	}

	var val string
	err := json.Unmarshal(data, &val)
	*s = EnforcedString(val)
	return err
}

// MarshalJSON return json marshal
func (s *EnforcedString) MarshalJSON() ([]byte, error) {
	return json.Marshal(*s)
}

// APIErrType is a field containing more specific API error types
// that may be checked by the receiver.
type APIErrType string

const (
	// APIErrTypeUnknown is for API errors that are not strongly
	// typed.
	APIErrTypeUnknown APIErrType = "unknown"

	// APIErrTypeInvalidGrant corresponds with Keycloak's
	// OAuthErrorException due to "invalid_grant".
	APIErrTypeInvalidGrant = "oauth: invalid grant"
)

// ParseAPIErrType is a convenience method for returning strongly
// typed API errors.
func ParseAPIErrType(err error) APIErrType {
	if err == nil {
		return APIErrTypeUnknown
	}
	switch {
	case strings.Contains(err.Error(), "invalid_grant"):
		return APIErrTypeInvalidGrant
	default:
		return APIErrTypeUnknown
	}
}

// APIError holds message and statusCode for api errors
type APIError struct {
	Code    int        `json:"code"`
	Message string     `json:"message"`
	Type    APIErrType `json:"type"`
}

// Error stringifies the APIError
func (apiError APIError) Error() string {
	return apiError.Message
}

type CreateCustomerRequest struct {
	BusName            string  `json:"BusName"`
	ProvinceID         *int64  `json:"ProvinceId"`
	CityID             *int64  `json:"CityId"`
	Email              *string `json:"Email"`
	Mobile             *string `json:"Mobile"`
	PhoneNo            *string `json:"PhoneNo"`
	Name               *string `json:"Name"`
	Family             *string `json:"Family"`
	NationalCode       *int64  `json:"NationalCode"`
	BirthDate          *string `json:"BirthDate"`
	Sexuality          *string `json:"Sexuality"`
	RealOrFinancial    *int64  `json:"RealOrFinancial"`
	Address            *string `json:"Address"`
	FinCode            *int64  `json:"FinCode"`
	IDNo               *int64  `json:"IDNo"`
	RegisterNumber     *int64  `json:"RegisterNumber"`
	BusinessCategoryID *int64  `json:"BusinessCategoryId"`
}

type CreateCustomerResponse struct {
	BusinessCode string `json:"BussinesCode"`
	BusinessID   string `json:"BussinessID"`
	Existed      bool   `json:"Existed"`
}

type CreateTransactionRequest struct {
	Data   Data                `json:"Data"`
	Items  []map[string]*int64 `json:"Items"`
	AddSub []AddSub            `json:"AddSub"`
}

type AddSub struct {
	AddSubID  int64 `json:"AddSubID"`
	TASAmount int64 `json:"TASAmount"`
}

type Data struct {
	TransactionID        interface{} `json:"TransactionID"`
	BusinessID           int64       `json:"BusinessID"`
	DocAliasID           int64       `json:"DocAliasId"`
	TransStateID         int64       `json:"TransStateId"`
	FactorTypeID         int64       `json:"FactorTypeId"`
	CalcTaxAndToll       int64       `json:"CalcTaxAndToll"`
	TransDiscountAmount  int64       `json:"TransDiscountAmount"`
	TransDiscountPercent float64     `json:"TransDiscountPercent"`
	DepartmentID         int64       `json:"DepartmentID"`
	SettlementID         int64       `json:"SettlementID"`
	Description          string      `json:"Description"`
}

type GetCustomerResponse struct {
	Data  []Datum2    `json:"data"`
	Error interface{} `json:"error"`
}

type Datum2 struct {
	RowNumber           string `json:"RowNumber"`
	BusinessID          string `json:"BusinessID"`
	BusinessCode        string `json:"BusinessCode"`
	BusinessName        string `json:"BusinessName"`
	Address             string `json:"Address"`
	PhoneNo             string `json:"PhoneNo"`
	FinCode             string `json:"FinCode"`
	Mobile              string `json:"Mobile"`
	Fax                 string `json:"Fax"`
	PriceLevelID        string `json:"PriceLevelID"`
	DefaultDiscount     int64  `json:"DefaultDiscount"`
	BusinessCategoryID  string `json:"BusinessCategoryID"`
	AccID               string `json:"AccID"`
	PostalCode          string `json:"PostalCode"`
	GeoRegionID         string `json:"GeoRegionID"`
	DeliveryRegionID    string `json:"DeliveryRegionID"`
	DefaultSettlementID string `json:"DefaultSettlementID"`
	WithoutCredit       string `json:"WithoutCredit"`
	County              string `json:"County"`
	RegisterNumber      string `json:"RegisterNumber"`
	LatinName           string `json:"LatinName"`
	BusinessActivity    string `json:"BusinessActivity"`
	BusDescription      string `json:"BusDescription"`
	InActive            string `json:"InActive"`
	Name                string `json:"Name"`
	Family              string `json:"Family"`
	FatherName          string `json:"FatherName"`
	NationalCode        string `json:"NationalCode"`
	IDNo                string `json:"IDNo"`
	//BirthDate           *CustomTime `json:"BirthDate"`
	BirthPlace        string      `json:"BirthPlace"`
	BankID            string      `json:"BankID"`
	AccountType       string      `json:"AccountType"`
	AccountNo         string      `json:"AccountNo"`
	Sexuality         string      `json:"Sexuality"`
	Creditable        string      `json:"Creditable"`
	ProvinceID        string      `json:"ProvinceID"`
	CityID            string      `json:"CityID"`
	TaxCityCode       string      `json:"TaxCityCode"`
	TaxProvincesCode  string      `json:"TaxProvincesCode"`
	PerCityCode       string      `json:"PerCityCode"`
	Email             string      `json:"Email"`
	WebSite           string      `json:"WebSite"`
	RelatedUserID     string      `json:"RelatedUserID"`
	CreatorUserID     string      `json:"Creator_UserID"`
	CreationDate      *CustomTime `json:"Creation_Date"`
	CardNumber        string      `json:"CardNumber"`
	CardSerial        string      `json:"CardSerial"`
	RepresentorCode   string      `json:"RepresentorCode"`
	RepresentorID     string      `json:"RepresentorID"`
	CheckCredit       int64       `json:"CheckCredit"`
	UnCashCredit      int64       `json:"UnCashCredit"`
	ModificationDate  *CustomTime `json:"Modification_Date"`
	IsCustomer        string      `json:"IsCustomer"`
	IsVendor          string      `json:"IsVendor"`
	IsSaleManager     string      `json:"IsSaleManager"`
	IsRepresentor     string      `json:"IsRepresentor"`
	IsDeliveryManager string      `json:"IsDeliveryManager"`
	RealOrFinancial   string      `json:"RealOrFinancial"`
}

type CreateTransactionResponse struct {
	Data  []Datum     `json:"data"`
	Error interface{} `json:"error"`
}

type Datum struct {
	TransactionID int64 `json:"TransactionID"`
	TransNumber   int64 `json:"TransNumber"`
	TransLineID   int64 `json:"TransLineID"`
	ItemID        int64 `json:"ItemID"`
}

type CreateServiceRequest struct {
	ServiceName    string `json:"ServiceName"`
	ServiceCode    string `json:"ServiceCode"`
	ItemCategoryID int64  `json:"ItemCategoryID"`
	IAGroupID      int64  `json:"IAGroupID"`
}

type CreateServiceResponse struct {
	ServiceName    string `json:"ServiceName"`
	ItemCategoryID int64  `json:"ItemCategoryId"`
}

type CustomTime struct {
	time.Time
}

func (ct *CustomTime) UnmarshalJSON(data []byte) error {
	str := strings.Trim(string(data), `"`)
	if str == "" {
		return nil
	}

	t, err := time.Parse("2006-01-02 15:04:05", str)
	if err != nil {
		return fmt.Errorf("failed to parse time: %w", err)
	}

	ct.Time = t
	return nil
}
