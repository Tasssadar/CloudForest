package CloudForest

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"math/rand"
	"strconv"
	"strings"
)

//Forest represents a collection of decision trees grown to predict Target.
type Forest struct {
	//Forest string
	Target string
	//Ntrees int
	//Categories string
	//Shrinkage int*/
	Trees []*Tree
}

//BUG(ryan) GrowRandomForest is a stub... need to decide what all paramters to expose and then implment using Tree.Grow
func GrowRandomForest(fm *FeatureMatrix, target *Feature, nSamples int, mTry int, nTrees int, leafSize int) (f *Forest) {
	f = &Forest{target.Name, make([]*Tree, 0, nTrees)}

	canidates := make([]int, 0, len(fm.Data))
	targeti := fm.Map[target.Name]
	for i := 0; i < len(fm.Data); i++ {
		if i != targeti {
			canidates = append(canidates, i)
		}
	}
	for i := 0; i < nTrees; i++ {
		//sample nCases case with replacment
		//BUG...abstract randdom sampleing and make sure it is good enough
		//fmt.Println("Tree ", i)
		cases := make([]int, 0, nSamples)
		nCases := len(fm.Data[0].Missing)
		for i := 0; i < nSamples; i++ {
			cases = append(cases, rand.Intn(nCases))
		}

		f.Trees = append(f.Trees, &Tree{&Node{nil, nil, "", nil}})
		f.Trees[i].Grow(fm, target, cases, canidates, mTry, leafSize)
	}
	return
}

/*SavePredictor save's a fprest in rf-ace's "stoicastic forest" sf format
It won't include fields that are not use by cloud forest.
Start of an example file:

	FOREST=RF,TARGET="N:CLIN:TermCategory:NB::::",NTREES=12800,CATEGORIES="",SHRINKAGE=0
	TREE=0
	NODE=*,PRED=3.48283,SPLITTER="B:SURV:Family_Thyroid:F::::maternal",SPLITTERTYPE=CATEGORICAL,LVALUES="false",RVALUES="true"
	NODE=*L,PRED=3.75
	NODE=*R,PRED=1

Node should be a path the form *LRL where * indicates the root L and R indicate Left and Right.*/
func (f *Forest) SavePredictor(w io.Writer) {
	fmt.Fprintf(w, "FOREST=RF,TARGET=%v,NTREES=%v\n", f.Target, len(f.Trees))
	for i, t := range f.Trees {
		fmt.Fprintf(w, "TREE=%v\n", i)
		t.Root.Write(w, "*")
	}

}

/*ParseRfAcePredictor reads a forest from an io.Reader.
The forest should be in rf-ace's "stoicastic forest" sf format
It ignores fields that are not use by cloud forest.
Start of an example file:

	FOREST=RF,TARGET="N:CLIN:TermCategory:NB::::",NTREES=12800,CATEGORIES="",SHRINKAGE=0
	TREE=0
	NODE=*,PRED=3.48283,SPLITTER="B:SURV:Family_Thyroid:F::::maternal",SPLITTERTYPE=CATEGORICAL,LVALUES="false",RVALUES="true"
	NODE=*L,PRED=3.75
	NODE=*R,PRED=1

Node should be a path the form *LRL where * indicates the root L and R indicate Left and Right.*/
func ParseRfAcePredictor(input io.Reader) *Forest {
	r := bufio.NewReader(input)
	var forest *Forest
	var tree *Tree
	for {
		line, err := r.ReadString('\n')
		if err == io.EOF {
			break
		}
		parsed := ParseRfAcePredictorLine(line)
		switch {
		case strings.HasPrefix(line, "FOREST"):
			forest = new(Forest)
			forest.Target = parsed["TARGET"]

		case strings.HasPrefix(line, "TREE"):
			tree = new(Tree)
			forest.Trees = append(forest.Trees, tree)

		case strings.HasPrefix(line, "NODE"):
			var splitter *Splitter

			pred := ""
			if filepred, ok := parsed["PRED"]; ok {
				pred = filepred
			}

			if stype, ok := parsed["SPLITTERTYPE"]; ok {
				splitter = new(Splitter)
				splitter.Feature = parsed["SPLITTER"]
				switch stype {
				case "CATEGORICAL":
					splitter.Numerical = false

					splitter.Left = make(map[string]bool)
					for _, f := range strings.Split(parsed["LVALUES"], ":") {
						splitter.Left[f] = true
					}

					splitter.Right = make(map[string]bool)
					for _, f := range strings.Split(parsed["RVALUES"], ":") {
						splitter.Right[f] = true
					}

				case "NUMERICAL":
					splitter.Numerical = true
					lvalue, err := strconv.ParseFloat(parsed["LVALUES"], 64)
					if err != nil {
						log.Print("Error parsing lvalues value ", err)
					}
					splitter.Value = float64(lvalue)
				}
			}

			tree.AddNode(parsed["NODE"], pred, splitter)

		}
	}

	return forest

}

/*ParseRfAcePredictorLine parses a single line of an rf-ace sf "stoicastic forest"
and returns a map[string]string of the key value pairs
Some examples of valid input lines:

	FOREST=RF,TARGET="N:CLIN:TermCategory:NB::::",NTREES=12800,CATEGORIES="",SHRINKAGE=0
	TREE=0
	NODE=*,PRED=3.48283,SPLITTER="B:SURV:Family_Thyroid:F::::maternal",SPLITTERTYPE=CATEGORICAL,LVALUES="false",RVALUES="true"
	NODE=*L,PRED=3.75
	NODE=*R,PRED=1
*/
func ParseRfAcePredictorLine(line string) map[string]string {
	clauses := make([]string, 0)
	insidequotes := make([]string, 0)
	terms := strings.Split(strings.TrimSpace(line), ",")
	for _, term := range terms {
		term = strings.TrimSpace(term)
		quotes := strings.Count(term, "\"")
		//if quotes have been opend join terms
		if quotes == 1 || len(insidequotes) > 0 {
			insidequotes = append(insidequotes, term)
		} else {
			//If the term doesn't have an = in it join it to the last term
			if strings.Count(term, "=") == 0 {
				clauses[len(clauses)-1] += "," + term
			} else {
				clauses = append(clauses, term)
			}
		}
		//quotes were closed
		if quotes == 1 && len(insidequotes) > 1 {
			clauses = append(clauses, strings.Join(insidequotes, ","))
			insidequotes = make([]string, 0)
		}

	}
	parsed := make(map[string]string, 0)
	for _, clause := range clauses {
		vs := strings.Split(clause, "=")
		for i, v := range vs {
			vs[i] = strings.Trim(strings.TrimSpace(v), "\"")
		}
		if len(vs) != 2 {
			fmt.Println("Parser Choked on : \"", line, "\"")
		}
		parsed[vs[0]] = vs[1]
	}

	return parsed
}
