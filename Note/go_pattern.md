

# Transfer from c++ to go

Go is a systems programming language intended to be a general-purpose systems language, like C++. These are some notes on Go for experienced C++ programmers. This document discusses the differences between Go and C++, and says little to nothing about the similarities.

For a more general introduction to Go, see the [Go tutorial](https://www.cs.cmu.edu/afs/cs.cmu.edu/academic/class/15440-f11/go/doc/go_tutorial.html) and [Effective Go](https://www.cs.cmu.edu/afs/cs.cmu.edu/academic/class/15440-f11/go/doc/effective_go.html).

For a detailed description of the Go language, see the [Go spec](https://www.cs.cmu.edu/afs/cs.cmu.edu/academic/class/15440-f11/go/doc/go_spec.html).

## Conceptual Differences

- Go does not have classes with constructors or destructors. Instead of class methods, a class inheritance hierarchy, and virtual functions, Go provides *interfaces*, which are [discussed in more detail below](https://www.cs.cmu.edu/afs/cs.cmu.edu/academic/class/15440-f11/go/doc/go_for_cpp_programmers.html#Interfaces). Interfaces are also used where C++ uses templates.
- Go uses garbage collection. It is not necessary (or possible) to release memory explicitly. The garbage collection is (intended to be) incremental and highly efficient on modern processors.
- Go has pointers but not pointer arithmetic. You cannot use a pointer variable to walk through the bytes of a string.
- Arrays in Go are first class values. When an array is used as a function parameter, the function receives **a copy of the array**, not a pointer to it. However, in practice functions often use slices for parameters; slices hold pointers to underlying arrays. Slices are [discussed further below](https://www.cs.cmu.edu/afs/cs.cmu.edu/academic/class/15440-f11/go/doc/go_for_cpp_programmers.html#Slices).
- Strings are provided by the language. They may not be changed once they have been created.
- Hash tables are provided by the language. They are called **maps**.
- Separate threads of execution, and communication channels between them, are provided by the language. This is [discussed further below](https://www.cs.cmu.edu/afs/cs.cmu.edu/academic/class/15440-f11/go/doc/go_for_cpp_programmers.html#Goroutines).
- Certain types (**maps and channels**, described further below) are passed by **reference**, not by value. That is, passing a map to a function does not copy the map, and if the function changes the map the change will be seen by the caller. In C++ terms, one can think of these as being reference types.
- Go does not use header files. Instead, each source file is part of a defined ***package***. When a package defines an object (type, constant, variable, function) with **a name starting with an upper case letter, that object is visible to any other file which imports that package.**
- Go does not support implicit type conversion. Operations that mix different types require casts (called conversions in Go).
- Go does not support function overloading and does not support user defined operators.
- Go does not support `const` or `volatile` qualifiers.
- Go uses `nil` for invalid pointers, where C++ uses `NULL` or simply `0`.

## Syntax

The declaration syntax is reversed compared to C++. You write the name followed by the type. Unlike in C++, the syntax for a type does not match the way in which the variable is used. Type declarations may be read easily from left to right.

```
Go                           C++
var v1 int                // int v1;
var v2 string             // const std::string v2;  (approximately)
var v3 [10]int            // int v3[10];
var v4 []int              // int* v4;  (approximately)
var v5 struct { f int }   // struct { int f; } v5;
var v6 *int               // int* v6;  (but no pointer arithmetic)
var v7 map[string]int     // unordered_map<string, int>* v7;  (approximately)
var v8 func(a int) int    // int (*v8)(int a);
```

Declarations generally take the form of a keyword followed by the name of the object being declared. The keyword is one of `var`, `func`, `const`, or `type`. Method declarations are a minor exception in that the receiver appears before the name of the object being declared; see the [discussion of interfaces](https://www.cs.cmu.edu/afs/cs.cmu.edu/academic/class/15440-f11/go/doc/go_for_cpp_programmers.html#Interfaces).

You can also use a keyword followed by a series of declarations in parentheses.

```
var (
    i int
    m float64
)
```

When declaring a function, you must either provide a name for each parameter or not provide a name for any parameter; you can't omit some names and provide others. You may group several names with the same type:

```
func f(i, j, k int, s, t string)
```

A variable may be initialized when it is declared. When this is done, specifying the type is permitted but not required. When the type is not specified, the type of the variable is the type of the initialization expression.

```
var v = *p
```

See also the [discussion of constants, below](https://www.cs.cmu.edu/afs/cs.cmu.edu/academic/class/15440-f11/go/doc/go_for_cpp_programmers.html#Constants). If a variable is not initialized explicitly, the type must be specified. In that case it will be implicitly initialized to the type's zero value (0, nil, etc.). There are no uninitialized variables in Go.

Within a function, a short declaration syntax is available with `:=` .

```
v1 := v2
```

This is equivalent to

```
var v1 = v2
```

Go permits multiple assignments, which are done in parallel.

```
i, j = j, i    // Swap i and j.
```

Functions may have multiple return values, indicated by a list in parentheses. The returned values can be stored by assignment to a list of variables.

```
func f() (i int, j int) { ... }
v1, v2 = f()
```

Go code uses very few semicolons in practice. Technically, all Go statements are terminated by a semicolon. However, Go treats the end of a non-blank line as a semicolon unless the line is clearly incomplete (the exact rules are in [the language specification](https://www.cs.cmu.edu/afs/cs.cmu.edu/academic/class/15440-f11/go/doc/go_spec.html#Semicolons)). A consequence of this is that in some cases Go does not permit you to use a line break. For example, you may not write

```
func g()
{                  // INVALID
}
func g(){          // VALID
}

```

A semicolon will be inserted after `g()`, causing it to be a function declaration rather than a function definition. Similarly, you may not write

```
if x {
}				   //;
else {             // INVALID
}

if x {
}else {             // VALID
}
```

A semicolon will be inserted after the `}` preceding the `else`, causing a syntax error.

Since semicolons do end statements, you may continue using them as in C++. However, that is not the recommended style. Idiomatic Go code omits unnecessary semicolons, which in practice is all of them other than the initial `for` loop clause and cases where you want several short statements on a single line.

While we're on the topic, we recommend that rather than worry about semicolons and brace placement, you format your code with the `gofmt` program. That will produce a single standard Go style, and let you worry about your code rather than your formatting. While the style may initially seem odd, it is as good as any other style, and familiarity will lead to comfort.

When using a pointer to a struct, you use `.` instead of `->`. Thus syntactically speaking a structure and a pointer to a structure are used in the same way.

```
type myStruct struct { i int }
var v9 myStruct              // v9 has structure type
var p9 *myStruct             // p9 is a pointer to a structure
f(v9.i, p9.i)
```

Go does not require parentheses around the condition of a `if` statement, or the expressions of a `for` statement, or the value of a `switch` statement. On the other hand, it does require curly braces around the body of an `if` or `for` statement.

```
if a < b { f() }             // Valid
if (a < b) { f() }           // Valid (condition is a parenthesized expression)
if (a < b) f()               // INVALID
for i = 0; i < 10; i++ {}    // Valid
for (i = 0; i < 10; i++) {}  // INVALID
```

Go does not have a `while` statement nor does it have a `do/while` statement. The `for` statement may be used with a single condition, which makes it equivalent to a `while` statement. Omitting the condition entirely is an endless loop.

Go permits `break` and `continue` to specify a label. The label must refer to a `for`, `switch`, or `select` statement.

In a `switch` statement, `case` labels do not fall through. You can make them fall through using the `fallthrough` keyword. This applies even to adjacent cases.

```
switch i {
case 0:  // empty case body
case 1:
    f()  // f is not called when i == 0!
}
```

But a `case` can have multiple values.

```
switch i {
case 0, 1:
    f()  // f is called if i == 0 || i == 1.
}
```

The values in a `case` need not be constants—or even integers; any type that supports the equality comparison operator, such as strings or pointers, can be used—and if the `switch` value is omitted it defaults to `true`.

```
switch {
case i < 0:
    f1()
case i == 0:
    f2()
case i > 0:
    f3()
}
```

The `++` and `--` operators may only be used in statements, not in expressions. You cannot write `c = *p++`. `*p++` is parsed as `(*p)++`.

The `defer` statement may be used to call a function after the function containing the `defer` statement returns.

```
fd := open("filename")
defer close(fd)         // fd will be closed when this function returns.
```

## Constants

In Go constants may be *untyped*. This applies even to constants named with a `const` declaration, if no type is given in the declaration and the initializer expression uses only untyped constants. A value derived from an untyped constant becomes typed when it is used within a context that requires a typed value. This permits constants to be used relatively freely without requiring general implicit type conversion.

```
var a uint
f(a + 1)  // untyped numeric constant "1" becomes typed as uint
```

The language does not impose any limits on the size of an untyped numeric constant or constant expression. A limit is only applied when a constant is used where a type is required.

```
const huge = 1 << 100
f(huge >> 98)
```

Go does not support enums. Instead, you can use the special name `iota` in a single `const` declaration to get a series of increasing value. When an initialization expression is omitted for a `const`, it reuses the preceding expression.

```
const (
    red = iota   // red == 0
    blue         // blue == 1
    green        // green == 2
)
```

## Slices

A slice is conceptually a struct with three fields: a pointer to an array, a length, and a capacity. Slices support the `[]` operator to access elements of the underlying array. The builtin `len` function returns the length of the slice. The builtin `cap` function returns the capacity.

Given an array, or another slice, a new slice is created via `a[I:J]`. This creates a new slice which refers to `a`, starts at index `I`, and ends before index `J`. It has length `J - I`. The new slice refers to the same array to which `a` refers. That is, changes made using the new slice may be seen using `a`. The capacity of the new slice is simply the capacity of `a` minus `I`. The capacity of an array is the length of the array. You may also assign an array pointer to a variable of slice type; given `var s []int; var a[10] int`, the assignment `s = &a` is equivalent to `s = a[0:len(a)]`.

What this means is that Go uses slices for some cases where C++ uses pointers. If you create a value of type `[100]byte` (an array of 100 bytes, perhaps a buffer) and you want to pass it to a function without copying it, you should declare the function parameter to have type `[]byte`, and pass the address of the array. Unlike in C++, it is not necessary to pass the length of the buffer; it is efficiently accessible via `len`.

The slice syntax may also be used with a string. It returns a new string, whose value is a substring of the original string. Because strings are immutable, string slices can be implemented without allocating new storage for the slices's contents.

## Making values

Go has a builtin function `new` which takes a type and allocates space on the heap. The allocated space will be zero-initialized for the type. For example, `new(int)` allocates a new int on the heap, initializes it with the value `0`, and returns its address, which has type `*int`. Unlike in C++, `new` is a function, not an operator; `new int` is a syntax error.

Map and channel values must be allocated using the builtin function `make`. A variable declared with map or channel type without an initializer will be automatically initialized to `nil`. Calling `make(map[int]int)` returns a newly allocated value of type `map[int]int`. Note that `make` returns a value, not a pointer. This is consistent with the fact that map and channel values are passed by reference. Calling `make` with a map type takes an optional argument which is the expected capacity of the map. Calling `make` with a channel type takes an optional argument which sets the buffering capacity of the channel; the default is 0 (unbuffered).

The `make` function may also be used to allocate a slice. In this case it allocates memory for the underlying array and returns a slice referring to it. There is one required argument, which is the number of elements in the slice. A second, optional, argument is the capacity of the slice. For example, `make([]int, 10, 20)`. This is identical to `new([20]int)[0:10]`. Since Go uses garbage collection, the newly allocated array will be discarded sometime after there are no references to the returned slice.

## Interfaces

Where C++ provides classes, subclasses and templates, Go provides interfaces. A Go interface is similar to a C++ pure abstract class: a class with no data members, with methods which are all pure virtual. However, in Go, any type which provides the methods named in the interface may be treated as an implementation of the interface. No explicitly declared inheritance is required. The implementation of the interface is entirely separate from the interface itself.

A method looks like an ordinary function definition, except that it has a *receiver*. The receiver is similar to the `this` pointer in a C++ class method.

```
type myType struct { i int }
func (p *myType) get() int { return p.i }
```

This declares a method `get` associated with `myType`. The receiver is named `p` in the body of the function.

Methods are defined on named types. If you convert the value to a different type, the new value will have the methods of the new type, not the old type.

You may define methods on a builtin type by declaring a new named type derived from it. The new type is distinct from the builtin type.

```
type myInteger int
func (p myInteger) get() int { return int(p) } // Conversion required.
func f(i int) { }
var v myInteger
// f(v) is invalid.
// f(int(v)) is valid; int(v) has no defined methods.
```

Given this interface:

```
type myInterface interface {
	get() int
	set(i int)
}
```

we can make `myType` satisfy the interface by adding

```
func (p *myType) set(i int) { p.i = i }
```

Now any function which takes `myInterface` as a parameter will accept a variable of type `*myType`.

```
func getAndSet(x myInterface) {}
func f1() {
	var p myType
	getAndSet(&p)
}
```

In other words, if we view `myInterface` as a C++ pure abstract base class, defining `set` and `get` for `*myType` made `*myType` automatically inherit from `myInterface`. A type may satisfy multiple interfaces.

An anonymous field may be used to implement something much like a C++ child class.

```
type myChildType struct { myType; j int }
func (p *myChildType) get() int { p.j++; return p.myType.get() }
```

This effectively implements `myChildType` as a child of `myType`.

```
func f2() {
	var p myChildType
	getAndSet(&p)
}
```

The `set` method is effectively inherited from `myChildType`, because methods associated with the anonymous field are promoted to become methods of the enclosing type. In this case, because `myChildType` has an anonymous field of type `myType`, the methods of `myType` also become methods of `myChildType`. In this example, the `get` method was overridden, and the `set` method was inherited.

This is not precisely the same as a child class in C++. When a method of an anonymous field is called, its receiver is the field, not the surrounding struct. In other words, methods on anonymous fields are not virtual functions. When you want the equivalent of a virtual function, use an interface.

A variable which has an interface type may be converted to have a different interface type using a special construct called a type assertion. This is implemented dynamically at run time, like C++ `dynamic_cast`. Unlike `dynamic_cast`, there does not need to be any declared relationship between the two interfaces.

```
type myPrintInterface interface {
  print()
}
func f3(x myInterface) {
	x.(myPrintInterface).print()  // type assertion to myPrintInterface
}
```

The conversion to `myPrintInterface` is entirely dynamic. It will work as long as the underlying type of x (the *dynamic type*) defines a `print` method.

Because the conversion is dynamic, it may be used to implement generic programming similar to templates in C++. This is done by manipulating values of the minimal interface.

```
type Any interface { }
```

Containers may be written in terms of `Any`, but the caller must unbox using a type assertion to recover values of the contained type. As the typing is dynamic rather than static, there is no equivalent of the way that a C++ template may inline the relevant operations. The operations are fully type-checked at run time, but all operations will involve a function call.

```
type iterator interface {
	get() Any
	set(v Any)
	increment()
	equal(arg *iterator) bool
}
```

## Goroutines

Go permits starting a new thread of execution (a *goroutine*) using the `go` statement. The `go` statement runs a function in a different, newly created, goroutine. All goroutines in a single program share the same address space.

Internally, goroutines act like coroutines that are multiplexed among multiple operating system threads. You do not have to worry about these details.

```
func server(i int) {
    for {
        print(i)
        sys.sleep(10)
    }
}
go server(1)
go server(2)
```

(Note that the `for` statement in the `server` function is equivalent to a C++ `while (true)` loop.)

Goroutines are (intended to be) cheap.

Function literals (which Go implements as closures) can be useful with the `go` statement.

```
var g int
go func(i int) {
	s := 0
	for j := 0; j < i; j++ { s += j }
	g = s
}(1000)  // Passes argument 1000 to the function literal.
```

## Channels

Channels are used to communicate between goroutines. Any value may be sent over a channel. Channels are (intended to be) efficient and cheap. To send a value on a channel, use `<-` as a binary operator. To receive a value on a channel, use `<-` as a unary operator. When calling functions, channels are passed by reference.

The Go library provides mutexes, but you can also use a single goroutine with a shared channel. Here is an example of using a manager function to control access to a single value.

```
type cmd struct { get bool; val int }
func manager(ch chan cmd) {
	var val int = 0
	for {
		c := <- ch
		if c.get { c.val = val; ch <- c }
		else { val = c.val }
	}
}
```

In that example the same channel is used for input and output. This is incorrect if there are multiple goroutines communicating with the manager at once: a goroutine waiting for a response from the manager might receive a request from another goroutine instead. A solution is to pass in a channel.

```
type cmd2 struct { get bool; val int; ch <- chan int }
func manager2(ch chan cmd2) {
	var val int = 0
	for {
		c := <- ch
		if c.get { c.ch <- val }
		else { val = c.val }
	}
}
```

To use `manager2`, given a channel to it:

```
func f4(ch <- chan cmd2) int {
	myCh := make(chan int)
	c := cmd2{ true, 0, myCh }   // Composite literal syntax.
	ch <- c
	return <-myCh
}
```

```go
package main
import "fmt"
func main(){
	fmt.Println("hellp")
}
```



# Tour to go

### Variables

```go
//Inside a function, the := short assignment statement can be used in place of a var declaration with implicit type.
//Outside a function, every statement begins with a keyword (var, func, and so on) and so the := construct is not available.
var x,y int=1,2
var(
	x int=1
	y int=2
)
func foo(){x:=1}
bool
string
int  int8  int16  int32  int64
uint uint8 uint16 uint32 uint64 uintptr
byte // alias for uint8
rune // alias for int32 represents a Unicode code point
float32 float64
complex64 complex128

fmt.Println(x)
fmt.Printf("%T,%v\\n",x,x)
//%T type。
//%v value

//zero value
0 : numeric
false : boolean
"" : strings.
//The expression T(v) converts the value v to the type T.
var i int = 42
var f float64 = float64(i)
var u uint = uint(f)

i := 42
f := float64(i)
u := uint(f)
```

### Constant

Constants are declared like variables, but with the `const` keyword.

Constants cannot be declared using the `:=` syntax.

```go
const Pi=3.14
//runtime.GOARCH 
//runtime.GOOS darwin,linux
switch os:=runtime.GOOS;os{...}
```

### Deferred function

Deferred function calls are pushed onto a stack. When a function returns, its deferred calls are executed in last-in-first-out order.

```go
defer func1()
defer func2()
func3()
return
//func3 -> func2 -> func1
```

### Pointer & Struct

```go
var p *int //zero value is nil
p:=&i //p is pointer
*p=21

type Vertex struct{
	x int
	y int
}
v:=Vertex{1,2}
v.x=3
p=&v
p.x=4
```

### Array & Slices

```go
//var a [n]type
var a []int
primes:=[6]int{2,3,5,7,11,13}
var s []int = primes[1:4]
s:=primes[1:4]
//Changing the elements of a slice modifies the corresponding elements of its underlying array.
s := []struct {
		i int
		b bool
	}{
		{2, true},
		{3, false},
	}
```

s[a:b]这里改变底层s的只有a，会把前面部分drop掉

The `make` function allocates a zeroed array and returns a slice that refers to that array:

```go
a := make([]int, 5)  // len(a)=5
```

To specify a capacity, pass a third argument to `make`:

```go
b := make([]int, 0, 5) // len(b)=0, cap(b)=5

b = b[:cap(b)] // len(b)=5, cap(b)=5
b = b[1:]      // len(b)=4, cap(b)=4
s = append(s, 2, 3, 4)
```

### Range

```go
for i,v:=range a{...}
//range return two arguments:index and value
func Pic(dx, dy int) [][]uint8 {
    p := make([][]uint8, dy)
    for i := range p {
        p[i] = make([]uint8, dx)
        for j := 0; j < dx; j++ {
           p[i][j] = uint8((i + j) / 2)
        }
    }
    return p
}
```

### Map

```go
myMap:=make(map[T1]T2)
myMap := map[string]int{"a": 1, "b": 2, "c": 3}Insert or update an element in map `m`:
```

Delete an element:

```go
delete(m, key)
```

Test that a key is present with a two-value assignment:

```go
elem, ok = m[key]
//if empty then ok is false, else true
```

### Function

**`fn func(float64, float64) float64`** is a function type declaration. It declares a function type named **`fn`** that takes two **`float64`** parameters and returns a **`float64`**

```go
func compute(x,y float64,fn func(float64, float64) float64) float64 {
	return fn(x, y)
}
```

**Function closures**

**`func adder() func(int) int`**: 这是函数签名，表示 **`adder`** 函数不接受任何参数，返回一个函数，该函数接受一个 **`int`** 参数并返回一个 **`int`**。

```go
func adder() func(int) int {
	sum := 0
	return func(x int) int {
		sum += x
		return sum
	}
}

func main() {
	pos, neg := adder(), adder()
	for i := 0; i < 10; i++ {
		fmt.Println(
			pos(i),
			neg(-2*i),
		)
	}
}
```

### Methods & Interface

```go
func (v Vertex) Abs() float64 {
	return math.Sqrt(v.X*v.X + v.Y*v.Y)
}
func (v *Vertex) Scale(f float64) {
	v.X = v.X * f
	v.Y = v.Y * f
}
v.Scale(10)
v.Abs()
p := &v
p.Scale(10) // OK

func Abs(v Vertex) float64 {
	return math.Sqrt(v.X*v.X + v.Y*v.Y)
}
func Scale(v *Vertex, f float64) {
	v.X = v.X * f
	v.Y = v.Y * f
}
Scale(&v, 10)
Abs(v)
```

**Interface !**

```go
type I interface {
	M()
}
type T struct {
	S string
}
// This method means type T implements the interface I,
// but we don't need to explicitly declare that it does so.
func (t T) M() {
	fmt.Println(t.S)
}
func main() {
	var i I = T{"hello"}
	i.M()
}
var i interface{}
//can give different types!
t := i.(T)
var i interface{} = "hello"
s, ok := i.(string) //ok is true
f, ok := i.(float64)//ok is false,f is 0

switch v := i.(type){...}
```

### Stringer & Error

**`fmt.Println`** 函数在打印值时会首先检查该值是否实现了 **`Stringer`** 接口。如果实现了 **`Stringer`** 接口，那么 **`fmt.Println`** 会调用该类型的 **`String()`** 方法来获取该值的字符串表示形式。

One of the most ubiquitous interfaces is `[Stringer](<https://go.dev/pkg/fmt/#Stringer>)` defined by the `[fmt](<https://go.dev/pkg/fmt/>)` package.

```go
type Stringer interface {
    String() string
}
```

A `Stringer` is a type that can describe itself as a string.

```go
type Person struct {
	Name string
	Age  int
}

func (p Person) String() string {
	return fmt.Sprintf("%v (%v years)", p.Name, p.Age)
}

func main() {
	a := Person{"Arthur Dent", 42}
	z := Person{"Zaphod Beeblebrox", 9001}
	fmt.Println(a, z)
}
type MyError struct {
	When time.Time
	What string
}

func (e *MyError) Error() string {
	return fmt.Sprintf("at %v, %s",
		e.When, e.What)
}
```

### IO

The `io.Reader` interface has a `Read` method:

```
func (T) Read(b []byte) (n int, err error)
```

`Read` populates the given byte slice with data and returns the number of bytes populated and an error value. It returns an `io.EOF` error when the stream ends.

```go
r := strings.NewReader("Hello, Reader!")
//creates a strings.Reader
b := make([]byte, 8)
for {
	n, err := r.Read(b)
	fmt.Printf("n = %v err = %v b = %v\\n", n, err, b)
	fmt.Printf("b[:n] = %q\\n", b[:n])
	if err == io.EOF {
		break
	}
}
```

**`strings`** 包中，**`NewReader`** 函数接受一个字符串作为参数，并返回一个 **`\*strings.Reader`** 类型的值。**`strings.Reader`** 类型实现了 **`io.Reader`** 接口

```go
type MyReader struct{}

// TODO: Add a Read([]byte) (int, error) method to MyReader.
func (m MyReader) Read(b []byte) (int,error){
	for i := range b {
        b[i] = 'A'
    }
    return len(b), nil
}
func main() {
	reader.Validate(MyReader{})
}
```

### Slice Sort & Sort

```go
slices.SortFunc(pairs, func(p, q pair) int { return q.x + q.y - p.x - p.y })
```

```go
import "sort"
name1:=[]string{...}
sort.Strings(name1)
ints:=[]int{}
sort.Ints(ints)

```



## Concurrency

**Goroutines**

```go
go f(x, y, z)
//starts a new goroutine running
//just like if(fork()==0){f(x,y,z);} in linux
```

**Channel**

```go
ch := make(chan int)
ch1 := make(chan int, 100)//buffer channel, size of 100(can store 100 int data) and because channel, it's First In First Out(FIFO)
//Like maps and slices, channels must be created before use:
ch <- v    // Send v to channel ch.
v := <-ch  // Receive from ch, and
           // assign value to v.
//(The data flows in the direction of the arrow.)
//just like pipe in linux kernel
fmt.Println(<-ch)

v, ok := <-ch//if ch is closed, then ok is false
```

By default, sends and receives block until the other side is ready. This allows goroutines to synchronize without explicit locks or condition variables.

```go
func fibonacci(n int, c chan int) {
	x, y := 0, 1
	for i := 0; i < n; i++ {
		c <- x
		x, y = y, x+y
	}
	close(c)
}

func main() {
	c := make(chan int, 10)
	go fibonacci(cap(c), c)
//receives values from the channel repeatedly until it is closed.
	for i := range c {
		fmt.Println(i)
	}
}
```

**Select**

A `select` blocks until one of its cases can run, then it executes that case. It chooses one at random if multiple are ready.

`select` 是与 `switch` 相似的控制结构，与 `switch` 不同的是，`select` 中虽然也有多个 `case`，但是这些 `case` 中的表达式必须都是 Channel 的收发操作， `case <-chan:`这种！而不是`case a:` ！

1. **`func() {...}`** 是一个匿名函数的定义，表示一个没有参数的函数。
2. **`()`** 表示对这个匿名函数的调用。通过 **`go`** 关键字，这个调用会在一个新的 Goroutine 中执行，实现并发执行。

```go
func fibonacci(c, quit chan int) {
	x, y := 0, 1
	for {
		select {
		case c <- x:
			x, y = y, x+y
		case <-quit:
			fmt.Println("quit")
			return
		}
	}
}

func main() {
	c := make(chan int)
	quit := make(chan int)
	go func() {
		for i := 0; i < 10; i++ {
			fmt.Println(<-c)
		}
		quit <- 0
	}()
	fibonacci(c, quit)
}
```

The `default` case in a `select` is run if no other case is ready.

Use a `default` case to try a send or receive without blocking:

```go
select {
case i := <-c:
    // use i
default:
    // receiving from c would block
}
```

**Mutex**

```go
import("syn")
type SafeCounter struct {
	mu sync.Mutex
	v  map[string]int
}
func (c *SafeCounter) Inc(key string) {
	c.mu.Lock()
	// Lock so only one goroutine at a time can access the map c.v.
	c.v[key]++
	c.mu.Unlock()
}

// Value returns the current value of the counter for the given key.
func (c *SafeCounter) Value(key string) int {
	c.mu.Lock()
	// Lock so only one goroutine at a time can access the map c.v.
	defer c.mu.Unlock()
	return c.v[key]
}
```

**Sync**

```go
var wg sync.WaitGroup
wg.Add(1)//increment the internal counter by 1.
defer wg.Done()//decrement the internal counter by 1.
wg.Wait()//In the main Goroutine (or any controlling Goroutine), block until the counter becomes 0

// SafeCache is a safe cache for visited URLs.
type SafeCache struct {
	mu    sync.Mutex
	cache map[string]bool
}

var cache = SafeCache{cache: make(map[string]bool)}

// Crawl uses fetcher to recursively crawl
// pages starting with url, to a maximum of depth.
func Crawl(url string, depth int, fetcher Fetcher) {
	if depth <= 0 {
		return
	}

	cache.mu.Lock()
	visited := cache.cache[url]
	cache.mu.Unlock()

	if visited {
		return
	}

	body, urls, err := fetcher.Fetch(url)
	cache.mu.Lock()
	cache.cache[url] = true
	cache.mu.Unlock()

	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("found: %s %q\\n", url, body)
	var wg sync.WaitGroup

	for _, u := range urls {
		wg.Add(1)
		go func(u string) {
			defer wg.Done()
			Crawl(u, depth-1, fetcher)
		}(u)
	}

	wg.Wait()
	return
}
```

## Abstract Syntax Tree、AST

词法分析会返回一个不包含空格、换行等字符的 Token 序列，例如：`package`, `json`, `import`, `(`, `io`, `)`, …，而语法分析会把 Token 序列转换成有意义的结构体，即语法树：

```go
"json.go": SourceFile {
    PackageName: "json",
    ImportDecl: []Import{
        "io",
    },
    TopLevelDecl: ...
}
```

# Patterns and Hints for Concurrency in Go

Concurrency is about dealing with lots of things at once. 

Parallelism is about doing lots of things at once.

### Pattern #1 Publish/subscribe server

Hint: Close a channel to signal  that no more values will be sent

```go
package main

type PubSub interface {
	// Publish publishes the event e to
	// all current subscriptions.
	Publish(e Event)
	// Subscribe registers c to receive future events.
	// All subscribers receive events in the same order,
	// and that order respects program order:
	// if Publish(e1) happens before Publish(e2),
	// subscribers receive e1 before e2.
	Subscribe(c chan<- Event)
	// Cancel cancels the prior subscription of channel c.
	// After any pending already-published events
	// have been sent on c, the server will signal that the
	// subscription is cancelled by closing c.
	Cancel(c chan<- Event)
}

```

Hint: Prefer defer for unlocking mutexes.

```go
package main

type Server struct {
	mu  sync.Mutex
	sub map[chan<- Event]bool
}

func (s *Server) Init() {
	s.sub = make(map[chan<- Event]bool)
}
func (s *Server) Publish(e Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for c := range s.sub {
		c <- e
	}
}
func (s *Server) Subscribe(c chan<- Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sub[c] {
		panic("pubsub: already subscribed")
	}
	s.sub[c] = true
}
func (s *Server) Cancel(c chan<- Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.sub[c] {
		panic("pubsub: not subscribed")
	}
	close(c)
	delete(s.sub, c)
}

```

Hint: Consider the effect  of slow goroutines

* Slow down event generation. 
* Drop events. 
  Examples: os/signal, runtime/pprof 
* Queue an arbitrary number of events

```go
package main

type Server struct {
	publish   chan Event
	subscribe chan subReq
	cancel    chan subReq
}
type subReq struct {
	c  chan<- Event
	ok chan bool
}

func (s *Server) Init() {
	s.publish = make(chan Event)
	s.subscribe = make(chan subReq)
	s.cancel = make(chan subReq)
	go s.loop()
}
func (s *Server) Publish(e Event) {
	s.publish <- e
}
func (s *Server) Subscribe(c chan<- Event) {
	r := subReq{c: c, ok: make(chan bool)}
	s.subscribe <- r
	if !<-r.ok {
		panic("pubsub: already subscribed")
	}
}
func (s *Server) Cancel(c chan<- Event) {
	r := subReq{c: c, ok: make(chan bool)}
	s.cancel <- r
	if !<-r.ok {
		panic("pubsub: not subscribed")
	}
}
func (s *Server) loop() {
	sub := make(map[chan<- Event]bool)
	for {
		select {
		case e := <-s.publish:
			for c := range sub {
				c <- e
			}
		case r := <-s.subscribe:
			if sub[r.c] {
				r.ok <- false
				break
			}
			sub[r.c] = true
			r.ok <- true
		case c := <-s.cancel:
			if !sub[r.c] {
				r.ok <- false
				break
			}
			close(r.c)
			delete(sub, r.c)
			r.ok <- true
		}
	}
}

```

Hint: Convert mutexes  into goroutines  when it makes programs clearer

#### v1

```go
// <-chan receive only, from chan send to
// chan<- send only, send to chan
func helper(in <-chan Event,
	out chan<- Event) {
	var q []Event
	for {
		select {
		case e := <-in:
			q = append(q, e)
		case out <- q[0]:
			q = q[1:]
		}
	}
}
```

#### v2

```go
func helper(in <-chan Event,
	out chan<- Event) {
	var q []Event
	for {
		// Decide whether and what to send.
		var sendOut chan<- Event
		var next Event
		if len(q) > 0 {
			sendOut = out
			next = q[0]
		}
		select {
		case e := <-in:
			q = append(q, e)
		case sendOut <- next:
			q = q[1:]
		}
	}
}

```

#### v3

```go
func helper(in <-chan Event,
	out chan<- Event) {
	var q []Event
	for in != nil || len(q) > 0 {
		// Decide whether and what to send.
		var sendOut chan<- Event
		var next Event
		if len(q) > 0 {
			sendOut = out
			next = q[0]
		}
		select {
		case e, ok := <-in:
			if !ok {
				in = nil // stop receiving from in
				break
			}
			q = append(q, e)
		case sendOut <- next:
			q = q[1:]
		}
	}
	close(out)
}

```

Hint: Use goroutines }  to let independent concerns  run independently

```go
func (s *Server) loop() {
	sub := make(map[chan<- Event]chan<- Event)
	for {
		select {
		case e := <-s.publish:
			for _, h := range sub {
				h <- e
			}
		case r := <-s.subscribe:
			if sub[r.c] != nil {
				r.ok <- false
				break
			}
			h = make(chan Event)
			go helper(h, r.c)
			sub[r.c] = h
			r.ok <- true
		case c := <-s.cancel:
			if sub[r.c] == nil {
				r.ok <- false
				break
			}
			close(sub[r.c])
			delete(sub, r.c)
			r.ok <- true
		}
	}
}

```



1. Publish(e Event)
   Publish方法允许发布者发布事件e。这个事件将被发送到所有当前的订阅者。按照接口注释的描述，所有订阅者都应该以相同的顺序接收到事件，这个顺序遵循程序的执行顺序。这意味着如果Publish(e1)操作在Publish(e2)之前执行，那么所有的订阅者都应该首先接收到e1事件，然后才是e2事件。

2. Subscribe(c chan<- Event)
   Subscribe方法允许新的订阅者通过提供一个事件通道c来注册自己，以便于接收未来的事件。这个通道是单向的，只能用于发送事件（chan<- Event表示一个只能发送Event类型数据的通道）。
3. Cancel(c chan<- Event)
   Cancel方法允许订阅者取消之前的订阅。取消订阅后，任何已经发布但尚未发送到订阅者的事件会继续发送，直到所有挂起的事件都处理完毕。之后，为了通知订阅者订阅已经被取消，订阅者提供的通道c将被关闭。

发布者和订阅者不需要知道对方的存在。发布者只负责将消息发送到一个中介通道，而订阅者则从中介通道订阅消息。这种方式简化了系统的依赖关系，使得系统组件更易于管理和扩展。

发布/订阅模式广泛应用于构建松耦合、高可扩展和响应式的系统。典型的应用场景包括：

- 消息队列和事件总线系统
- 实时数据分发，如股票行情、新闻更新
- 微服务架构中的服务间通信
- 分布式系统中的事件驱动架构

### Pattern #2  Work scheduler

```go
func main(servers []string, numTask int,
	call func(srv string, task int)) {
	idle := make(chan string, len(servers))
	for _, srv := range servers {
		idle <- srv
	}
	for task := 0; task < numTask; task++ {
		task := task
		srv := <-idle
		go func() {
			call(srv, task)
			idle <- srv
		}()
	}
	for i := 0; i < len(servers); i++ {
		<-idle
	}
}

```

Hint: Think carefully before  introducing unbounded queuing.

Hint: Close a channel to signal  that no more values will be sent

Hint: Use goroutines  to let independent concerns  run independently.

#### v1

```go
func Schedule(servers []string, numTask int,
	call func(srv string, task int)) {
	work := make(chan int)
	done := make(chan bool)
	runTasks := func(srv string) {
		for task := range work {
			call(srv, task)
		}
		done <- true
	}
	for _, srv := range servers {
		go runTasks(srv)
	}
	for task := 0; task < numTask; task++ {
		work <- task
	}
	close(work)
	for i := 0; i < len(servers); i++ {
		<-done
	}
}

```

#### v2 with datarace

```go
func Schedule(servers chan string, numTask int,
	call func(srv string, task int)) {
	work := make(chan int)
	done := make(chan bool)
	runTasks := func(srv string) {
		for task := range work {
			call(srv, task)
		}
		done <- true
	}
	go func() {
		for _, srv := range servers {
			go runTasks(srv)
		}
	}()
	for task := 0; task < numTask; task++ {
		work <- task
	}
	close(work)
    // done is to make sure every goroutines exit properly
	for i := 0; i < len(servers); i++ {
		<-done
	}
}
```

#### v3

```go
func Schedule(servers chan string, numTask int,
	call func(srv string, task int) bool) {
	work := make(chan int, numTask)
	done := make(chan bool)
	runTasks := func(srv string) {
		for task := range work {
			if call(srv, task) {
				done <- true
			} else {
				work <- task
			}
		}
	}
	go func() {
		for _, srv := range servers {
			go runTasks(srv)
		}
	}()
	for task := 0; task < numTask; task++ {
		work <- task
	}
	for i := 0; i < numTask; i++ {
		<-done
	}
	close(work)
}
```

```go
go func() { 
	for { 
		select { 
		case srv := <-servers: 
			go runTasks(srv) 
		case <-exit: 
			return 
		} 
	} 
}() 
```

### Pattern #3  Replicated service client

```go
import "type"

type ReplicatedClient interface {
	// Init initializes the client to use the given servers.
	// To make a particular request later,
	// the client can use callOne(srv, args), where srv
	// is one of the servers from the list.
	Init(servers []string, callOne func(string, Args) Reply)
	// Call makes a request on any available server.
	// Multiple goroutines may call Call concurrently.
	Call(args Args) Reply
}

type Client struct {
	servers []string
	callOne func(string, Args) Reply
	mu      sync.Mutex
	prefer  int
}

func (c *Client) Init(servers []string, callOne func(string, Args) Reply) {
	c.servers = servers
	c.callOne = callOne
}
func (c *Client) Call(args Args) Reply {
	type result struct {
		serverID int
		reply    Reply
	}

	const timeout = 1 * time.Second
	t := time.NewTimer(timeout)
	defer t.Stop()
	done := make(chan result, len(c.servers))

	for id := 0; id < len(c.servers); id++ {
		id := id
		go func() {
			done <- result{id, c.callOne(c.servers[id], args)}
		}()
		select {
		case r := <-done:
			return r.reply
		case <-t.C:
			// timeout
			t.Reset(timeout)
		}
	}
	r := <-done
	return r.reply
}

```

Hint: Use a goto if that is  the clearest way to write the code

```go
func (c *Client) Call(args Args) Reply {
	type result struct {
		serverID int
		reply    Reply
	}

	const timeout = 1 * time.Second
	t := time.NewTimer(timeout)
	defer t.Stop()
	done := make(chan result, len(c.servers))
	c.mu.Lock()
	prefer := c.prefer
	c.mu.Unlock()
	var r result
	for off := 0; off < len(c.servers); off++ {
		id := (prefer + off) % len(c.servers)
		go func() {
			done <- result{id, c.callOne(c.servers[id], args)}
		}()
		select {
		case r = <-done:
			goto Done
		case <-t.C:
			// timeout
			t.Reset(timeout)
		}
	}
	r = <-done
Done:
	c.mu.Lock()
	c.prefer = r.serverID
	c.mu.Unlock()
	return r.reply
}
```



### Pattern #4  Protocol multiplexer

```go
package main

type ProtocolMux interface {
	// Init initializes the mux to manage messages to the given service.
	Init(Service)
	// Call makes a request with the given message and returns the reply.
	// Multiple goroutines may call Call concurrently.
	Call(Msg) Msg
}
type Service interface {
	// ReadTag returns the muxing identifier in the request or reply message.
	// Multiple goroutines may call ReadTag concurrently.
	ReadTag(Msg) int64
	// Send sends a request message to the remote service.
	// Send must not be called concurrently with itself.
	Send(Msg)
	// Recv waits for and returns a reply message from the remote service.
	// Recv must not be called concurrently with itself.
	Recv() Msg
}

type Mux struct {
	srv     Service
	send    chan Msg
	mu      sync.Mutex
	pending map[int64]chan<- Msg
}

func (m *Mux) Init(srv Service) {
	m.srv = srv
	m.pending = make(map[int64]chan Msg)
	go m.sendLoop()
	go m.recvLoop()
}
func (m *Mux) sendLoop() {
	for args := range m.send {
		m.srv.Send(args)
	}
}
func (m *Mux) recvLoop() {
	for {
		reply := m.srv.Recv()
		tag := m.srv.ReadTag(reply)
		m.mu.Lock()
		done := m.pending[tag]
		delete(m.pending, tag)
		m.mu.Unlock()
		if done == nil {
			panic("unexpected reply")
		}
		done <- reply
	}
}
func (m *Mux) Call(args Msg) (reply Msg) {
	tag := m.srv.ReadTag(args)
	done := make(chan Msg, 1)
	m.mu.Lock()
	if m.pending[tag] != nil {
		m.mu.Unlock()
		panic("mux: duplicate call tag")
	}
	m.pending[tag] = done
	m.mu.Unlock()
	m.send <- args
	return <-done
}

```

### Hints

Use the race detector, for development and even production. 
Convert data state into code state when it makes programs clearer. 
Convert mutexes into goroutines when it makes programs clearer. 
Use additional goroutines to hold additional code state. 
Use goroutines to let independent concerns run independently. 
Consider the effect of slow goroutines. 
Know why and when each communication will proceed. 
Know why and when each goroutine will exit. 
Type Ctrl-\ to kill a program and dump all its goroutine stacks. 
Use the HTTP server’s /debug/pprof/goroutine to inspect live goroutine stacks. 
Use a buffered channel as a concurrent blocking queue. 
Think carefully before introducing unbounded queuing. 
Close a channel to signal that no more values will be sent. 
Stop timers you don’t need. 
Prefer defer for unlocking mutexes. 
Use a mutex if that is the clearest way to write the code. 
Use a goto if that is the clearest way to write the code. 
Use goroutines, channels, and mutexes together 
if that is the clearest way to write the code.

## Closure

followers vote for primary or leader

```go
package main
import "sync"
func main(){
	var wg sync.WaitGroup
	for i:=0;i<5;i++{
		wg.Add(1)
		go func(x int){
			sendRPC(x)
			wg.Done()
		}(i)
		// bag go!
		// go func(i int){
		// 	sendRPC(i)
		// 	wg.Done()
		// }()
	}
	wg.Wait()
}

func sendRPC(i int) {
	println(i)
}
```

## Format

```go
gofmt -w filename.go
// fotmat all .go file in current dir
gofmt -l -w .
goimports -w filename.go
```

