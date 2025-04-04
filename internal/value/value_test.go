package value

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestValueConstructor(t *testing.T) {
	t.Run("Null", func(t *testing.T) {
		require.Equal(t, Null(), Null())
		require.True(t, IsNull(Null()))
	})

	t.Run("Bool", func(t *testing.T) {
		require.Equal(t, True(), True())
		require.Equal(t, False(), False())
		require.NotEqual(t, True(), False())
		require.True(t, IsBool(Bool(true)))
	})

	t.Run("Num", func(t *testing.T) {
		require.Equal(t, Num(10), Num(10.0))
		require.NotEqual(t, Num(10.0), Num(10.1))
		require.True(t, IsNum(Num(10)))
	})

	t.Run("Str", func(t *testing.T) {
		require.Equal(t, Str("test"), Str("test"))
		require.NotEqual(t, Str("test"), Str("notest"))
		require.True(t, IsStr(Str("test")))
	})

	t.Run("Arr", func(t *testing.T) {
		require.True(t, IsArr(Arr(10, Num(20), Str("test"))))
	})

	t.Run("Obj", func(t *testing.T) {
		obj := Obj(map[string]any{
			"n": Num(10),
			"s": Str("test"),
			"b": True(),
			"a": Arr(1, "test"),
			"o": ObjValue{"id": Num(10), "name": Str("Object10")},
		})
		require.True(t, IsObj(obj))
	})
}

func TestValueNew(t *testing.T) {
	type myint int
	type myuint uint
	type myfloat float64
	type mybool bool
	type mystring string

	tests := []struct {
		name     string
		expected Value
		input    any
	}{
		{"Num from int", Num(1), 1},
		{"Num from float", Num(10), 10.0},
		{"Num from custom int", Num(10), myint(10)},
		{"Num from custom float", Num(10.1), myfloat(10.1)},
		{"Num from uint", Num(10), uint(10)},
		{"Num from custom uint", Num(10), myuint(10)},

		{"Bool from bool", True(), true},
		{"Bool from custom bool", False(), mybool(false)},

		{"Str from string", Str("test"), "test"},
		{"Str from custom String", Str("test"), mystring("test")},

		{"Arr from []any", ArrValue{Num(1), Str("test")}, []any{1, "test"}},
		{"Arr from []int", ArrValue{Num(1), Num(2), Num(3)}, []int{1, 2, 3}},

		{"Obj from map[string]any", ObjValue{"int": Num(1), "str": Str("test")}, map[string]any{"int": 1, "str": "test"}},
		{"Obj from map[string]int", ObjValue{"f1": Num(1), "f2": Num(2)}, map[string]int{"f1": 1, "f2": 2}},
		{"Deep object",
			ObjValue{
				"tags": Arr("user", "new", "google"),
				"user": ObjValue{
					"id":      Num(10),
					"emails":  Arr("email1", "email2"),
					"name":    Str("User1"),
					"country": ObjValue{"name": Str("USA"), "code": Str("us")}},
			},
			map[string]any{
				"user": map[string]any{
					"id":   10,
					"name": "User1",
					"country": map[string]string{
						"name": "USA",
						"code": "us",
					},
					"emails": []string{"email1", "email2"},
				},
				"tags": []any{"user", "new", "google"},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.expected, New(test.input))
		})
	}
}

func TestValueCast(t *testing.T) {
	tests := []struct {
		name     string
		expected Value
		input    Value
		vtype    ValueType
	}{
		// Analog !!arg JS expression
		{"Null to Bool", False(), Null(), BoolType},
		{"Bool to Bool", True(), True(), BoolType},
		{"Num to True", True(), Num(1), BoolType},
		{"Num to False", False(), Num(0), BoolType},
		{"Str to True", True(), Str("test"), BoolType},
		{"Str to False", False(), Str(""), BoolType},
		{"Arr To Bool", True(), ArrValue{}, BoolType},
		{"Obj To Bool", True(), ObjValue{}, BoolType},

		// Analog of arg * 1 JS expression
		{"Null to Num", Num(0), Null(), NumType},
		{"True to Num", Num(1), True(), NumType},
		{"False to Num", Num(0), False(), NumType},
		{"Num to Num", Num(10), Num(10), NumType},
		{"Empty Str to Num", Num(0), Str(""), NumType},
		{"Number Str to Num", Num(10), Str("10"), NumType},
		{"Number Str to Num 2", Num(10.1), Str("  10.1  "), NumType},
		{"Non number Str to Num", Null(), Str("bbb"), NumType},
		{"Empty Arr To Num", Num(0), Arr(), NumType},
		{"Arr with one elem to Num", Num(10), Arr("10"), NumType},
		{"Arr with non num elem to Num", Null(), Arr("bla"), NumType},
		{"Arr with many elems to Num", Null(), Arr(1, 2), NumType},
		{"Obj to Num", Null(), ObjValue{}, NumType},

		// Analog of arg + "" JS expression
		{"Null to Str", Str("null"), Null(), StrType},
		{"True to Str", Str("true"), True(), StrType},
		{"False to Str", Str("false"), False(), StrType},
		{"Number to Str", Str("10.1"), Num(10.1), StrType},
		{"Str to Str", Str("test"), Str("test"), StrType},
		{"Empty Arr to Str", Str(""), Arr(), StrType},
		{"Arr to Str", Str("1,2,3,test,,10,20"), Arr(1, 2, 3, "test", Arr(), Arr(10, 20)), StrType},
		{"Obj to Str", Null(), ObjValue{}, StrType},

		// No cast to Arr/Obj
		{"Null to Arr", Null(), Null(), ArrType},
		{"Bool to Arr", Null(), True(), ArrType},
		{"Num to Arr", Null(), Num(10), ArrType},
		{"Str to Arr", Null(), Str(""), ArrType},
		{"Arr to Arr", Arr(1, 2), Arr(1, 2), ArrType},
		{"Obj to Arr", Null(), ObjValue{}, ArrType},

		{"Null to Obj", Null(), Null(), ObjType},
		{"Bool to Obj", Null(), True(), ObjType},
		{"Num to Obj", Null(), Num(10), ObjType},
		{"Str to Obj", Null(), Str(""), ObjType},
		{"Arr to Obj", Null(), Arr(1, 2), ObjType},
		{"Obj to Obj", ObjValue{"f1": Num(10)}, ObjValue{"f1": Num(10)}, ObjType},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.expected, test.input.Cast(test.vtype))
		})
	}

}

func TestValueEqual(t *testing.T) {
	require.False(t, Equal(Str("10"), Num(10)), "Values of different types are not equal")
	require.True(t, Equal(Num(10), Num(10)))
	require.True(t, Equal(Arr(1, 2, "test"), Arr(1, 2, "test")))
	require.True(t, Equal(Arr(1, 2, Arr(10, 20)), Arr(1, 2, Arr(10, 20))))
	require.False(t, Equal(Arr(1, 2), Arr(2, 1)))
	require.True(t, Equal(ObjValue{"f1": Num(1), "f2": Arr(1, 2)}, ObjValue{"f1": Num(1), "f2": Arr(1, 2)}))
	require.False(t, Equal(ObjValue{"f1": Num(1)}, ObjValue{"f1": Num(1), "f2": Num(2)}))
}

func TestObjValuePath(t *testing.T) {
	obj := ObjValue{
		"user_id": Num(1),
		"user": ObjValue{
			"name":  Str("Bob"),
			"age":   Num(25),
			"admin": False(),
		},
		"country": ObjValue{
			"name": Str("USA"),
			"code": Str("us"),
		},
	}
	require.Equal(t, Str("Bob"), obj.Path("user", "name"))
	require.Equal(t, Num(25), obj.Path("user", "age"))
	require.Equal(t, Null(), obj.Path("user", "country"))
	require.Equal(t, Num(1), obj.Path("user_id"))
	path := []string{"country", "code"}
	require.Equal(t, Str("us"), obj.Path(path...))
}

func TestValueString(t *testing.T) {
	tests := []struct {
		v any
		s string
	}{
		{Null(), "null"},
		{true, "true"},
		{false, "false"},
		{100, "100"},
		{"ASDF", "ASDF"},
		{Arr(1, "T", true, Null(), Arr(1, 2)), "1,T,true,null,1,2"},
		{ObjValue{"f": New(10)}, "Object"},
	}
	for _, tt := range tests {
		require.Equal(t, tt.s, New(tt.v).String())
	}
}
