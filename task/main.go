package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/julienschmidt/httprouter"
)

// DB :
var DB *sql.DB

// InitDB initialises the database pools with
func InitDB(dbUser, dbPassword, dbHost, dbPort string) (Db *sql.DB, err error) {

	DB, err = sql.Open("mysql", dbUser+":"+dbPassword+"@tcp("+dbHost+":"+dbPort+")/test")
	if err != nil {
		return DB, err
	}

	return DB, nil
}

func main() {
	dbUser := "root"
	dbPassword := ""
	dbHost := "localhost"
	dbPort := "3306"
	DB, err := InitDB(dbUser, dbPassword, dbHost, dbPort)

	if err != nil {
		panic(err.Error())
	}

	defer DB.Close()

	router := httprouter.New()
	AddRouteHandlers(router)

	log.Fatal(http.ListenAndServe(":5000", router))

}

// AddRouteHandlers :
func AddRouteHandlers(router *httprouter.Router) {

	router.POST("/add-student", AddUpdateStudent)
	router.POST("/update-student", AddUpdateStudent)
	router.POST("/remove-student", RemoveStudent)
	router.GET("/student-list/:student_id", GetStudents)

}

//---------------------------- Start declaring user define sturct ----------

// JSONMessageContent :
type JSONMessageContent struct {
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message"`
}

// JSONWrappedContent :
type JSONWrappedContent struct {
	StatusCode int         `json:"statusCode"`
	Content    interface{} `json:"content"`
}

// StudentInfo :
type StudentInfo struct {
	ID      int64  `json:"id,omitempty"`
	Name    string `json:"name,omitempty"`
	PhoneNo string `json:"phone_no,omitempty"`
	Address string `json:"address,omitempty"`
}

//---------------------------- End declaring user define sturct ----------

//---------------------------- Start declaring functions regarding api response ----------

// below functions manage api response as per api behave

// Response :
func Response(w http.ResponseWriter, r *http.Request, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(JSONMessage(code, message))
}

// JSONMessage :
func JSONMessage(code int, msg string) []byte {
	jsonString := JSONMessageContent{
		StatusCode: code,
		Message:    msg,
	}

	//result, err := json.Marshal(jsonString)
	result, err := json.MarshalIndent(jsonString, "", "    ")
	if err != nil {
		fmt.Println(err)
	}

	return result
}

// JSONMessageWithObj :
func JSONMessageWithObj(code int, obj interface{}) []byte {
	jsonString := JSONWrappedContent{
		StatusCode: code,
		Content:    obj,
	}

	result, err := json.MarshalIndent(jsonString, "", "    ")
	if err != nil {
		fmt.Println(err)
	}
	return result
}

// ResponseJSONObject :
func ResponseJSONObject(w http.ResponseWriter, code int, obj interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(obj.([]byte))
}

//---------------------------- Start declaring functions regarding api response ----------

//---------------------------- Start API functions ----------

// AddUpdateStudent :
func AddUpdateStudent(w http.ResponseWriter, r *http.Request, p httprouter.Params) {

	if r.Body == nil {
		Response(w, r, 400, "Invalid request - please check your input data.")
		return
	}
	var objStudent StudentInfo
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&objStudent)
	if err != nil {
		Response(w, r, 400, "Invalid request - please check your input data.")
		return
	}

	// ------------------Start validate input data -------------
	if objStudent.Name == "" {
		Response(w, r, 400, "Name can't be blank.")
		return
	}

	if objStudent.PhoneNo == "" {
		Response(w, r, 400, "Phone number can't be blank.")
		return
	}

	_, errNo := strconv.Atoi(objStudent.PhoneNo)
	if errNo != nil {
		Response(w, r, 400, "Mobile number not correct please enter correct number.")
		return
	}

	if objStudent.Address == "" {
		Response(w, r, 400, "Adress can't be blank.")
		return
	}
	// ------------------End validate input data -------------

	// -------- Insertion data in DB ---------
	if objStudent.ID == 0 {
		errInsert := insertStudent(objStudent)
		if errInsert != nil {
			Response(w, r, 500, "There is an error while inserting Student."+errInsert.Error())
			return
		}
		Response(w, r, 200, "Student Added SuccessFully.")
		return
	} else {
		errInsert := updateStudent(objStudent)
		if errInsert != nil {
			Response(w, r, 500, "There is an error while updating Student."+errInsert.Error())
			return
		}
		Response(w, r, 200, "Student Update SuccessFully.")
		return
	}

}

// GetStudents :
func GetStudents(w http.ResponseWriter, r *http.Request, p httprouter.Params) {

	studentID := p.ByName("student_id")
	searchType := r.URL.Query().Get("search_type")
	searchValue := r.URL.Query().Get("search_value")
	if studentID == "" {
		Response(w, r, 400, "student id can't be blank.")
		return
	}

	objStudent, err := GetStudentList(studentID, searchType, searchValue)
	if err != nil {
		Response(w, r, 500, "Error while fatching student data from db.")
		return
	}

	final := JSONMessageWithObj(http.StatusOK, objStudent)
	ResponseJSONObject(w, http.StatusOK, final)
	return

}

// RemoveStudent :
func RemoveStudent(w http.ResponseWriter, r *http.Request, p httprouter.Params) {

	// ------------------check form Data -------------
	studentID := r.FormValue("student_id")
	if studentID == "" {
		Response(w, r, 400, "Student id can't blank.")
		return
	}

	_, err := strconv.Atoi(studentID)
	if err != nil {
		Response(w, r, 400, "Student id can't correct.")
		return
	}

	// -------- Remove data in DB ---------
	errRemove := removeStudent(studentID)
	if errRemove != nil {
		Response(w, r, 500, "There is an error while removing Student."+errRemove.Error())
		return
	}
	Response(w, r, 200, "Student Removed SuccessFully.")
	return

}

//---------------------------- End API functions ----------

// -------------------------- Start Data functions those functions insert/update/delete/fetch records from database ----------

// insertStudent :
func insertStudent(objStudent StudentInfo) error {

	var sqlStr string

	sql := "INSERT INTO tbl_student (display_name, name, phone_no, address) VALUES "

	sqlStr += fmt.Sprintf("('%v', '%v','%v','%v')", objStudent.Name, strings.ToLower(objStudent.Name), objStudent.PhoneNo, objStudent.Address)

	sql = sql + sqlStr
	stmt, err := DB.Prepare(sql)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec()
	if err != nil {
		return err
	}

	return nil
}

// updateStudent :
func updateStudent(objStudent StudentInfo) error {

	sqlStatement := `UPDATE tbl_student SET name = ?, display_name = ?, phone_no = ?, address = ? WHERE id = ?`
	_, err := DB.Exec(sqlStatement, strings.ToLower(objStudent.Name), objStudent.Name, objStudent.PhoneNo, objStudent.Address, objStudent.ID)
	if err != nil {
		return err
	}

	return nil
}

// GetStudentList :
func GetStudentList(studentID, searchType, searchValue string) ([]StudentInfo, error) {
	var objStudents []StudentInfo
	var searchCondition string
	if searchType == "name" {
		searchCondition = " AND name ='" + searchValue + "' "
	} else if searchType == "phone" {
		searchCondition = " AND phone_no ='" + searchValue + "' "
	}

	if studentID != "all" {
		searchCondition += " AND id=" + studentID
	}

	sqlStr := "SELECT id, display_name, phone_no, address FROM tbl_student where status = ? " + searchCondition

	rows, err := DB.Query(sqlStr, 1)
	if err != nil && err != sql.ErrNoRows {
		return objStudents, err
	}
	defer rows.Close()

	for rows.Next() {
		var singleStudent StudentInfo
		err := rows.Scan(
			&singleStudent.ID,
			&singleStudent.Name,
			&singleStudent.PhoneNo,
			&singleStudent.Address,
		)
		if err != nil {
			return objStudents, err
		}
		objStudents = append(objStudents, singleStudent)
	}
	return objStudents, nil

}

// removeStudent :
func removeStudent(studentID string) error {
	sqlQuery := "Delete FROM tbl_student WHERE id = " + studentID
	stmt, err := DB.Prepare(sqlQuery)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec()
	if err != nil {
		return err
	}

	return nil
}

// -------------------------- End Data functions those functions insert/update/delete/fetch records from database ----------
