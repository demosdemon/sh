// Copyright (c) 2016, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package sh

import (
	"bytes"
	"fmt"
	"strings"
)

var defaultPos = Pos{}

func nodeFirstPos(ns []Node) Pos {
	if len(ns) == 0 {
		return defaultPos
	}
	return ns[0].Pos()
}

func wordFirstPos(ws []Word) Pos {
	if len(ws) == 0 {
		return defaultPos
	}
	return ws[0].Pos()
}

// File is a shell program.
type File struct {
	Name string

	Stmts []Stmt
}

func (f File) String() string { return stmtJoinWithEnd(f.Stmts, false) }

// Node represents an AST node.
type Node interface {
	fmt.Stringer
	Pos() Pos
}

func stringerJoin(strs []fmt.Stringer, sep string) string {
	var b bytes.Buffer
	for i, s := range strs {
		if i > 0 {
			fmt.Fprint(&b, sep)
		}
		fmt.Fprint(&b, s)
	}
	return b.String()
}

func nodeJoin(ns []Node, sep string) string {
	var b bytes.Buffer
	for i, n := range ns {
		if i > 0 {
			fmt.Fprint(&b, sep)
		}
		fmt.Fprint(&b, n)
	}
	return b.String()
}

func stmtJoinWithEnd(stmts []Stmt, end bool) string {
	var b bytes.Buffer
	newline := false
	for i, s := range stmts {
		if newline {
			newline = false
			fmt.Fprintln(&b)
		} else if i > 0 {
			fmt.Fprint(&b, "; ")
		}
		fmt.Fprint(&b, s)
		newline = s.newlineAfter()
	}
	if newline && end {
		fmt.Fprintln(&b)
	}
	return b.String()
}

func stmtJoin(stmts []Stmt) string {
	return stmtJoinWithEnd(stmts, true)
}

func stmtList(stmts []Stmt) string {
	if len(stmts) == 0 {
		return fmt.Sprint(SEMICOLON, " ")
	}
	s := stmtJoin(stmts)
	if len(s) > 0 && s[len(s)-1] == '\n' {
		return " " + s
	}
	return fmt.Sprintf(" %s%s ", s, SEMICOLON)
}

func semicolonIfNil(s fmt.Stringer) string {
	if s == nil {
		return fmt.Sprint(SEMICOLON, " ")
	}
	return s.String()
}

func wordJoin(words []Word, sep string) string {
	ns := make([]Node, len(words))
	for i, w := range words {
		ns[i] = w
	}
	return nodeJoin(ns, sep)
}

type Stmt struct {
	Node
	Position   Pos
	Negated    bool
	Assigns    []Assign
	Redirs     []Redirect
	Background bool
}

func (s Stmt) String() string {
	var strs []fmt.Stringer
	if s.Negated {
		strs = append(strs, NOT)
	}
	for _, a := range s.Assigns {
		strs = append(strs, a)
	}
	if s.Node != nil {
		strs = append(strs, s.Node)
	}
	for _, r := range s.Redirs {
		strs = append(strs, r)
	}
	if s.Background {
		strs = append(strs, AND)
	}
	return stringerJoin(strs, " ")
}
func (s Stmt) Pos() Pos { return s.Position }

func (s Stmt) newlineAfter() bool {
	for _, r := range s.Redirs {
		if r.Op == SHL || r.Op == DHEREDOC {
			return true
		}
	}
	return false
}

type Assign struct {
	Append bool
	Name   Node
	Value  Word
}

func (a Assign) String() string {
	if a.Name == nil {
		return a.Value.String()
	}
	if a.Append {
		return fmt.Sprint(a.Name, "+=", a.Value)
	}
	return fmt.Sprint(a.Name, "=", a.Value)
}

type Redirect struct {
	OpPos Pos
	Op    Token
	N     Lit
	Word  Word
}

func (r Redirect) String() string {
	if strings.HasPrefix(r.Word.String(), "<") {
		return fmt.Sprint(r.N, r.Op.String(), " ", r.Word)
	}
	return fmt.Sprint(r.N, r.Op.String(), r.Word)
}

type Command struct {
	Args []Word
}

func (c Command) String() string { return wordJoin(c.Args, " ") }
func (c Command) Pos() Pos       { return wordFirstPos(c.Args) }

type Subshell struct {
	Lparen, Rparen Pos
	Stmts          []Stmt
}

func (s Subshell) String() string {
	if len(s.Stmts) == 0 {
		// A space in between to avoid confusion with ()
		return fmt.Sprint(LPAREN, RPAREN)
	}
	return fmt.Sprint(LPAREN, stmtJoin(s.Stmts), RPAREN)
}
func (s Subshell) Pos() Pos { return s.Lparen }

type Block struct {
	Lbrace, Rbrace Pos
	Stmts          []Stmt
}

func (b Block) String() string {
	return fmt.Sprint(LBRACE, stmtList(b.Stmts), RBRACE)
}
func (b Block) Pos() Pos { return b.Rbrace }

type IfStmt struct {
	If, Fi    Pos
	Cond      Node
	ThenStmts []Stmt
	Elifs     []Elif
	ElseStmts []Stmt
}

func (s IfStmt) String() string {
	var b bytes.Buffer
	fmt.Fprint(&b, IF, semicolonIfNil(s.Cond), THEN, stmtList(s.ThenStmts))
	for _, elif := range s.Elifs {
		fmt.Fprint(&b, elif)
	}
	if len(s.ElseStmts) > 0 {
		fmt.Fprint(&b, ELSE, stmtList(s.ElseStmts))
	}
	fmt.Fprint(&b, FI)
	return b.String()
}
func (s IfStmt) Pos() Pos { return s.If }

type StmtCond struct {
	Stmts []Stmt
}

func (s StmtCond) String() string { return stmtList(s.Stmts) }
func (s StmtCond) Pos() Pos       { return s.Stmts[0].Pos() }

type CStyleCond struct {
	Lparen, Rparen Pos
	Cond           Node
}

func (c CStyleCond) String() string {
	return fmt.Sprintf(" ((%s)); ", c.Cond)
}
func (c CStyleCond) Pos() Pos { return c.Lparen }

type Elif struct {
	Elif      Pos
	Cond      Node
	ThenStmts []Stmt
}

func (e Elif) String() string {
	return fmt.Sprint(ELIF, semicolonIfNil(e.Cond), THEN, stmtList(e.ThenStmts))
}

type WhileStmt struct {
	While, Done Pos
	Cond        Node
	DoStmts     []Stmt
}

func (w WhileStmt) String() string {
	return fmt.Sprint(WHILE, semicolonIfNil(w.Cond), DO, stmtList(w.DoStmts), DONE)
}
func (w WhileStmt) Pos() Pos { return w.While }

type UntilStmt struct {
	Until, Done Pos
	Cond        Node
	DoStmts     []Stmt
}

func (u UntilStmt) String() string {
	return fmt.Sprint(UNTIL, semicolonIfNil(u.Cond), DO, stmtList(u.DoStmts), DONE)
}
func (u UntilStmt) Pos() Pos { return u.Until }

type ForStmt struct {
	For, Done Pos
	Cond      Node
	DoStmts   []Stmt
}

func (f ForStmt) String() string {
	return fmt.Sprint(FOR, " ", f.Cond, "; ", DO, stmtList(f.DoStmts), DONE)
}
func (f ForStmt) Pos() Pos { return f.For }

type WordIter struct {
	Name Lit
	List []Word
}

func (w WordIter) String() string {
	if len(w.List) < 1 {
		return w.Name.String()
	}
	return fmt.Sprint(w.Name, IN, " ", wordJoin(w.List, " "))
}
func (w WordIter) Pos() Pos { return w.Name.Pos() }

type CStyleLoop struct {
	Lparen, Rparen   Pos
	Init, Cond, Post Node
}

func (c CStyleLoop) String() string {
	return fmt.Sprintf("((%s; %s; %s))", c.Init, c.Cond, c.Post)
}
func (c CStyleLoop) Pos() Pos { return c.Lparen }

type UnaryExpr struct {
	OpPos Pos
	Op    Token
	Post  bool
	X     Node
}

func (u UnaryExpr) String() string {
	if u.Post {
		return fmt.Sprint(u.X, "", u.Op)
	}
	return fmt.Sprint(u.Op, "", u.X)
}
func (u UnaryExpr) Pos() Pos { return u.OpPos }

type BinaryExpr struct {
	OpPos Pos
	Op    Token
	X, Y  Node
}

func (b BinaryExpr) String() string {
	if b.Op == COMMA {
		return fmt.Sprint(b.X, "", b.Op, b.Y)
	}
	return fmt.Sprint(b.X, b.Op, b.Y)
}
func (b BinaryExpr) Pos() Pos { return b.X.Pos() }

type FuncDecl struct {
	Position  Pos
	BashStyle bool
	Name      Lit
	Body      Stmt
}

func (f FuncDecl) String() string {
	if f.BashStyle {
		return fmt.Sprint(FUNCTION, f.Name, "() ", f.Body)
	}
	return fmt.Sprint(f.Name, "() ", f.Body)
}
func (f FuncDecl) Pos() Pos { return f.Position }

type Word struct {
	Parts []Node
}

func (w Word) String() string { return nodeJoin(w.Parts, "") }
func (w Word) Pos() Pos       { return nodeFirstPos(w.Parts) }

type Lit struct {
	ValuePos Pos
	Value    string
}

func (l Lit) String() string { return l.Value }
func (l Lit) Pos() Pos       { return l.ValuePos }

type SglQuoted struct {
	Quote Pos
	Value string
}

func (q SglQuoted) String() string { return `'` + q.Value + `'` }
func (q SglQuoted) Pos() Pos       { return q.Quote }

type Quoted struct {
	QuotePos Pos
	Quote    Token
	Parts    []Node
}

func (q Quoted) String() string {
	stop := q.Quote
	if stop == DOLLSQ {
		stop = SQUOTE
	} else if stop == DOLLDQ {
		stop = DQUOTE
	}
	return fmt.Sprint(q.Quote, nodeJoin(q.Parts, ""), stop)
}
func (q Quoted) Pos() Pos { return q.QuotePos }

type CmdSubst struct {
	Left, Right Pos
	Backquotes  bool
	Stmts       []Stmt
}

func (c CmdSubst) String() string {
	if c.Backquotes {
		return "`" + stmtJoin(c.Stmts) + "`"
	}
	return fmt.Sprint(DOLLAR, "", LPAREN, stmtJoin(c.Stmts), RPAREN)
}
func (c CmdSubst) Pos() Pos { return c.Left }

type ParamExp struct {
	Dollar        Pos
	Short, Length bool
	Param         Lit
	Ind           *Index
	Repl          *Replace
	Exp           *Expansion
}

func (p ParamExp) String() string {
	if p.Short {
		return fmt.Sprint(DOLLAR, "", p.Param)
	}
	var b bytes.Buffer
	fmt.Fprint(&b, "${")
	if p.Length {
		fmt.Fprint(&b, HASH)
	}
	fmt.Fprint(&b, p.Param)
	if p.Ind != nil {
		fmt.Fprint(&b, p.Ind)
	}
	if p.Repl != nil {
		fmt.Fprint(&b, p.Repl)
	}
	if p.Exp != nil {
		fmt.Fprint(&b, p.Exp)
	}
	fmt.Fprint(&b, "}")
	return b.String()
}
func (p ParamExp) Pos() Pos { return p.Dollar }

type Index struct {
	Word Word
}

func (i Index) String() string { return fmt.Sprintf("[%s]", i.Word) }

type Replace struct {
	All        bool
	Orig, With Word
}

func (r Replace) String() string {
	if r.All {
		return fmt.Sprintf("//%s/%s", r.Orig, r.With)
	}
	return fmt.Sprintf("/%s/%s", r.Orig, r.With)
}

type Expansion struct {
	Op   Token
	Word Word
}

func (e Expansion) String() string { return fmt.Sprint(e.Op.String(), e.Word) }

type ArithmExpr struct {
	Dollar, Rparen Pos
	X              Node
}

func (a ArithmExpr) String() string {
	if a.X == nil {
		return "$(())"
	}
	return fmt.Sprintf("$((%s))", a.X)
}
func (a ArithmExpr) Pos() Pos { return a.Dollar }

type ParenExpr struct {
	Lparen, Rparen Pos
	X              Node
}

func (p ParenExpr) String() string { return fmt.Sprintf("(%s)", p.X) }
func (p ParenExpr) Pos() Pos       { return p.Lparen }

type CaseStmt struct {
	Case, Esac Pos
	Word       Word
	List       []PatternList
}

func (c CaseStmt) String() string {
	var b bytes.Buffer
	fmt.Fprint(&b, CASE, c.Word, IN)
	for i, plist := range c.List {
		if i > 0 {
			fmt.Fprint(&b, ";;")
		}
		fmt.Fprint(&b, plist)
	}
	fmt.Fprint(&b, "; ", ESAC)
	return b.String()
}
func (c CaseStmt) Pos() Pos { return c.Case }

type PatternList struct {
	Patterns []Word
	Stmts    []Stmt
}

func (p PatternList) String() string {
	return fmt.Sprintf(" %s) %s", wordJoin(p.Patterns, " | "), stmtJoin(p.Stmts))
}

type DeclStmt struct {
	Declare Pos
	Local   bool
	Opts    []Word
	Assigns []Assign
}

func (d DeclStmt) String() string {
	var strs []fmt.Stringer
	if d.Local {
		strs = append(strs, LOCAL)
	} else {
		strs = append(strs, DECLARE)
	}
	for _, w := range d.Opts {
		strs = append(strs, w)
	}
	for _, a := range d.Assigns {
		strs = append(strs, a)
	}
	return stringerJoin(strs, " ")
}
func (d DeclStmt) Pos() Pos { return d.Declare }

type ArrayExpr struct {
	Lparen, Rparen Pos
	List           []Word
}

func (a ArrayExpr) String() string {
	return fmt.Sprint(LPAREN, wordJoin(a.List, " "), RPAREN)
}
func (a ArrayExpr) Pos() Pos { return a.Lparen }

type CmdInput struct {
	Lss, Rparen Pos
	Stmts       []Stmt
}

func (c CmdInput) String() string {
	return fmt.Sprint(LSS, "", LPAREN, stmtJoin(c.Stmts), RPAREN)
}
func (c CmdInput) Pos() Pos { return c.Lss }
